package container

import (
	"fmt"
	"net"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/conditions"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
)

type Creator struct {
	LinkFinder conditions.LinkFinder
	Executor   executor.Executor
}

var hostNamespace = namespace.NewNamespace("/proc/self/ns/net")

type CreatorConfig struct {
	BridgeName      string
	SandboxNsPath   string
	ContainerNsPath string
	ContainerID     string
	InterfaceName   string
	VNI             int
	IPAMResult      types.Result
}

func (c *Creator) Setup(config CreatorConfig) (models.Container, error) {
	vxlanName := fmt.Sprintf("vxlan%d", config.VNI)
	sandboxNS := namespace.NewNamespace(config.SandboxNsPath)
	containerNS := namespace.NewNamespace(config.ContainerNsPath)

	sandboxLinkName := config.ContainerID
	if len(sandboxLinkName) > 15 {
		sandboxLinkName = sandboxLinkName[:15]
	}

	var routeCommands []commands.Command
	for _, route := range config.IPAMResult.IP4.Routes {
		routeCommand := commands.AddRoute{
			Interface:   config.InterfaceName,
			Destination: route.Dst,
			Gateway:     route.GW,
		}

		if routeCommand.Gateway == nil {
			routeCommand.Gateway = config.IPAMResult.IP4.Gateway
		}

		routeCommands = append(routeCommands, routeCommand)
	}

	err := c.Executor.Execute(
		commands.All(
			commands.InNamespace{
				Namespace: sandboxNS,
				Command: commands.Unless{
					Condition: conditions.LinkExists{
						LinkFinder: c.LinkFinder,
						Name:       vxlanName,
					},
					Command: commands.All(
						commands.InNamespace{
							Namespace: hostNamespace,
							Command: commands.All(
								commands.CreateVxlan{
									Name: vxlanName,
									VNI:  config.VNI,
								},
								commands.SetLinkNamespace{
									Namespace: sandboxNS.Path(),
									Name:      vxlanName,
								},
							),
						},
						commands.SetLinkUp{
							LinkName: vxlanName,
						},
					),
				},
			},
			commands.InNamespace{
				Namespace: containerNS,
				Command: commands.Group(
					append(
						[]commands.Command{
							commands.CreateVeth{
								Name:     config.InterfaceName,
								PeerName: sandboxLinkName,
								MTU:      1450,
							},
							commands.SetLinkNamespace{
								Name:      sandboxLinkName,
								Namespace: sandboxNS.Path(),
							},
							commands.AddAddress{
								InterfaceName: config.InterfaceName,
								Address:       config.IPAMResult.IP4.IP,
							},
							commands.SetLinkUp{
								LinkName: config.InterfaceName,
							},
						},
						routeCommands...,
					),
				),
			},
			commands.InNamespace{
				Namespace: sandboxNS,
				Command: commands.All(
					commands.SetLinkUp{
						LinkName: sandboxLinkName,
					},
					commands.Unless{
						Condition: conditions.LinkExists{
							LinkFinder: c.LinkFinder,
							Name:       config.BridgeName,
						},
						Command: commands.All(
							commands.CreateBridge{
								Name: config.BridgeName,
							},
							commands.AddAddress{
								InterfaceName: config.BridgeName,
								Address: net.IPNet{
									IP:   config.IPAMResult.IP4.Gateway,
									Mask: config.IPAMResult.IP4.IP.Mask,
								},
							},
							commands.SetLinkUp{
								LinkName: config.BridgeName,
							},
						),
					},
					commands.SetLinkMaster{
						Master: config.BridgeName,
						Slave:  vxlanName,
					},
					commands.SetLinkMaster{
						Master: config.BridgeName,
						Slave:  sandboxLinkName,
					},
				),
			},
		),
	)
	if err != nil {
		return models.Container{}, err
	}

	getHardwareAddressCommand := &commands.GetHardwareAddress{
		LinkName: config.InterfaceName,
	}

	command := commands.InNamespace{
		Namespace: containerNS,
		Command:   getHardwareAddressCommand,
	}
	err = c.Executor.Execute(command)
	if err != nil {
		return models.Container{}, err
	}

	return models.Container{
		ID:  config.ContainerID,
		MAC: getHardwareAddressCommand.Result.String(),
		IP:  config.IPAMResult.IP4.IP.IP.String(),
	}, nil
}
