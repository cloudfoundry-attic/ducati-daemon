package container

import (
	"net"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/conditions"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
)

type CommandBuilder struct {
	SandboxRepo   namespace.Repository
	MissWatcher   watcher.MissWatcher
	HostNamespace namespace.Namespace
}

func (b *CommandBuilder) IdempotentlyCreateSandbox(sandboxName string) executor.Command {
	sandboxNSPath := b.SandboxRepo.PathOf(sandboxName)
	sandboxNS := namespace.NewNamespace(sandboxNSPath)

	return commands.Unless{
		Condition: conditions.NamespaceExists{
			Name:       sandboxName,
			Repository: b.SandboxRepo,
		},
		Command: commands.All(
			commands.CreateNamespace{
				Name:       sandboxName,
				Repository: b.SandboxRepo,
			},
			commands.StartMonitor{
				Watcher:   b.MissWatcher,
				Namespace: sandboxNS,
			},
		),
	}
}

func (b *CommandBuilder) IdempotentlyCreateVxlan(vxlanName string, vni int, sandboxName string) executor.Command {
	sandboxNSPath := b.SandboxRepo.PathOf(sandboxName)
	sandboxNS := namespace.NewNamespace(sandboxNSPath)

	return commands.InNamespace{
		Namespace: sandboxNS,
		Command: commands.Unless{
			Condition: conditions.LinkExists{
				Name: vxlanName,
			},
			Command: commands.All(
				commands.InNamespace{
					Namespace: b.HostNamespace,
					Command: commands.All(
						commands.CreateVxlan{
							Name: vxlanName,
							VNI:  vni,
						},
						commands.MoveLink{
							Namespace: sandboxNSPath,
							Name:      vxlanName,
						},
					),
				},
				commands.InNamespace{
					Namespace: sandboxNS,
					Command: commands.SetLinkUp{
						LinkName: vxlanName,
					},
				},
			),
		},
	}
}

func (b *CommandBuilder) AddRoutes(interfaceName string, ipConfig *types.IPConfig) executor.Command {
	var routeCommands []executor.Command
	for _, route := range ipConfig.Routes {
		routeCommand := commands.AddRoute{
			Interface:   interfaceName,
			Destination: route.Dst,
			Gateway:     route.GW,
		}

		if routeCommand.Gateway == nil {
			routeCommand.Gateway = ipConfig.Gateway
		}

		routeCommands = append(routeCommands, routeCommand)
	}

	return commands.All(routeCommands...)
}

func (b *CommandBuilder) SetupVeth(containerNS namespace.Namespace, sandboxLinkName string,
	containerLinkName string, address net.IPNet, sandboxName string, routeCommand executor.Command) executor.Command {

	sandboxNSPath := b.SandboxRepo.PathOf(sandboxName)

	return commands.InNamespace{
		Namespace: containerNS,
		Command: commands.Group(
			append(
				[]executor.Command{
					commands.CreateVeth{
						Name:     containerLinkName,
						PeerName: sandboxLinkName,
						MTU:      1450,
					},
					commands.MoveLink{
						Name:      sandboxLinkName,
						Namespace: sandboxNSPath,
					},
					commands.AddAddress{
						InterfaceName: containerLinkName,
						Address:       address,
					},
					commands.SetLinkUp{
						LinkName: containerLinkName,
					},
				},
				routeCommand,
			),
		),
	}
}

func (b *CommandBuilder) IdempotentlySetupBridge(vxlanName, sandboxLinkName, sandboxName string, bridgeName string, ipamResult types.Result) executor.Command {
	sandboxNSPath := b.SandboxRepo.PathOf(sandboxName)
	sandboxNS := namespace.NewNamespace(sandboxNSPath)

	return commands.InNamespace{
		Namespace: sandboxNS,
		Command: commands.All(
			commands.SetLinkUp{
				LinkName: sandboxLinkName,
			},
			commands.Unless{
				Condition: conditions.LinkExists{
					Name: bridgeName,
				},
				Command: commands.All(
					commands.CreateBridge{
						Name: bridgeName,
					},
					commands.AddAddress{
						InterfaceName: bridgeName,
						Address: net.IPNet{
							IP:   ipamResult.IP4.Gateway,
							Mask: ipamResult.IP4.IP.Mask,
						},
					},
					commands.SetLinkUp{
						LinkName: bridgeName,
					},
				),
			},
			commands.SetLinkMaster{
				Master: bridgeName,
				Slave:  vxlanName,
			},
			commands.SetLinkMaster{
				Master: bridgeName,
				Slave:  sandboxLinkName,
			},
		),
	}
}
