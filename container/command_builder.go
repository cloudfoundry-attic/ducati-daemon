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
	MissWatcher   watcher.MissWatcher
	HostNamespace namespace.Namespace
}

func (b *CommandBuilder) IdempotentlyCreateSandbox(sandboxName, vxlanName, dnsAddress string) executor.Command {
	return commands.Unless{
		Condition: conditions.SandboxExists{
			Name: sandboxName,
		},
		Command: commands.All(
			commands.CreateSandbox{
				Name: sandboxName,
			},
			commands.StartDNSServer{
				SandboxName:   sandboxName,
				ListenAddress: dnsAddress,
			},
		),
	}
}

func (b *CommandBuilder) IdempotentlyCreateVxlan(
	vxlanName string,
	vni int,
	sandboxName string,
	sandboxNS namespace.Namespace,
) executor.Command {

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
							Namespace: sandboxNS,
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
				commands.StartMonitor{
					HostNamespace: b.HostNamespace,
					Watcher:       b.MissWatcher,
					SandboxName:   sandboxName,
					VxlanLinkName: vxlanName,
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

func (b *CommandBuilder) SetupVeth(
	containerNS namespace.Namespace,
	sandboxLinkName string,
	containerLinkName string,
	address net.IPNet,
	sandboxNS namespace.Namespace,
	routeCommand executor.Command,
) executor.Command {
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
						Namespace: sandboxNS,
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

func (b *CommandBuilder) IdempotentlySetupBridge(
	vxlanName, sandboxLinkName, bridgeName string,
	sandboxNS namespace.Namespace,
	ipamResult *types.Result,
) executor.Command {
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
					commands.SetLinkMaster{
						Master: bridgeName,
						Slave:  commands.DNS_INTERFACE_NAME,
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
