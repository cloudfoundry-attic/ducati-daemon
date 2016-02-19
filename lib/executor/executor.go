package executor

import (
	"fmt"
	"net"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/conditions"
	exec "github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/links"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
)

type Executor struct {
	Executor    exec.Executor
	LinkFactory LinkFactory
}

//go:generate counterfeiter --fake-name LinkFactory . LinkFactory
type LinkFactory interface {
	Exists(name string) bool
}

const selfPath = "/proc/self/ns/net"

func (e *Executor) EnsureVxlanDeviceExists(vni int, sandboxNS namespace.Namespace) (string, error) {
	vxlanName := fmt.Sprintf("vxlan%d", vni)

	hostNamespace := namespace.NewNamespace(selfPath)

	command := commands.InNamespace{
		Namespace: sandboxNS,
		Command: commands.Unless{
			Condition: conditions.LinkExists{
				LinkFinder: e.LinkFactory,
				Name:       vxlanName,
			},
			Command: commands.InNamespace{
				Namespace: hostNamespace,
				Command: commands.All(
					commands.CreateVxlan{
						Name: vxlanName,
						VNI:  vni,
					},
					commands.SetLinkNamespace{
						Namespace: sandboxNS.Path(),
						Name:      vxlanName,
					},
				),
			},
		},
	}

	err := e.Executor.Execute(command)
	if err != nil {
		return "", fmt.Errorf("failed to setup vxlan device: %s", err)
	}

	return vxlanName, nil
}

func (e *Executor) SetupSandboxNS(
	vxlanName, bridgeName string,
	sandboxNS namespace.Namespace,
	sandboxLinkName string,
	ipamResult types.Result,
) error {
	return e.Executor.Execute(
		commands.InNamespace{
			Namespace: sandboxNS,
			Command: commands.All(
				commands.SetLinkUp{
					LinkName: vxlanName,
				},
				commands.SetLinkUp{
					LinkName: sandboxLinkName,
				},
				commands.Unless{
					Condition: conditions.LinkExists{
						LinkFinder: e.LinkFactory,
						Name:       bridgeName,
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
		},
	)
}

func (e *Executor) SetupContainerNS(
	sandboxNsPath string,
	containerNsPath string,
	containerID string,
	interfaceName string,
	ipamResult types.Result,
) (string, string, error) {
	if len(containerID) > 11 {
		containerID = containerID[:11]
	}

	containerCommands := []commands.Command{
		commands.CreateVeth{
			Name:     interfaceName,
			PeerName: containerID,
			MTU:      links.VxlanVethMTU,
		},
		commands.SetLinkNamespace{
			Name:      containerID,
			Namespace: sandboxNsPath,
		},
		commands.AddAddress{
			InterfaceName: interfaceName,
			Address:       ipamResult.IP4.IP,
		},
		commands.SetLinkUp{
			LinkName: interfaceName,
		},
	}

	for _, route := range ipamResult.IP4.Routes {
		routeCommand := commands.AddRoute{
			Interface:   interfaceName,
			Destination: route.Dst,
			Gateway:     route.GW,
		}

		if routeCommand.Gateway == nil {
			routeCommand.Gateway = ipamResult.IP4.Gateway
		}

		containerCommands = append(containerCommands, routeCommand)
	}

	cmd := commands.InNamespace{
		Namespace: namespace.NewNamespace(containerNsPath),
		Command:   commands.All(containerCommands...),
	}

	err := e.Executor.Execute(cmd)
	if err != nil {
		return "", "", fmt.Errorf("container namespace setup failed: %s", err)
	}

	getHardwareAddressCommand := &commands.GetHardwareAddress{
		LinkName: interfaceName,
	}

	cmd = commands.InNamespace{
		Namespace: namespace.NewNamespace(containerNsPath),
		Command:   getHardwareAddressCommand,
	}
	err = e.Executor.Execute(cmd)
	if err != nil {
		return "", "", fmt.Errorf("failed to get container hardware address: %s", err)
	}

	return containerID, getHardwareAddressCommand.Result.String(), nil
}
