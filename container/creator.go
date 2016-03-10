package container

import (
	"fmt"
	"net"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/conditions"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
)

//go:generate counterfeiter -o ../fakes/command_builder.go --fake-name CommandBuilder . commandBuilder
type commandBuilder interface {
	IdempotentlyCreateSandbox(sandboxName string) commands.Command
	IdempotentlyCreateVxlan(vxlanName string, vni int, sandboxName string) commands.Command
	AddRoutes(interfaceName string, ipConfig *types.IPConfig) commands.Command
	SetupVeth(containerNS namespace.Namespace, sandboxLinkName string, containerLinkName string, address net.IPNet, sandboxName string, routeCommand commands.Command) commands.Command
	IdempotentlySetupBridge(vxlanName, sandboxLinkName, sandboxName string, bridgeName string, ipamResult types.Result) commands.Command
}

type Creator struct {
	LinkFinder     conditions.LinkFinder
	Executor       executor.Executor
	SandboxRepo    namespace.Repository
	Locker         commands.Locker
	Watcher        watcher.MissWatcher
	CommandBuilder commandBuilder
}

type CreatorConfig struct {
	NetworkID       string
	BridgeName      string
	ContainerNsPath string
	ContainerID     string
	InterfaceName   string
	HostIP          string
	VNI             int
	IPAMResult      types.Result
}

func (c *Creator) Setup(config CreatorConfig) (models.Container, error) {
	vxlanName := fmt.Sprintf("vxlan%d", config.VNI)
	sandboxName := fmt.Sprintf("vni-%d", config.VNI)
	containerNS := namespace.NewNamespace(config.ContainerNsPath)
	sandboxLinkName := config.ContainerID
	if len(sandboxLinkName) > 15 {
		sandboxLinkName = sandboxLinkName[:15]
	}

	var routeCommands = c.CommandBuilder.AddRoutes(config.InterfaceName, config.IPAMResult.IP4)

	c.Locker.Lock(sandboxName)
	defer c.Locker.Unlock(sandboxName)

	err := c.Executor.Execute(
		commands.All(
			c.CommandBuilder.IdempotentlyCreateSandbox(sandboxName),
			c.CommandBuilder.IdempotentlyCreateVxlan(vxlanName, config.VNI, sandboxName),
			c.CommandBuilder.SetupVeth(containerNS, sandboxLinkName, config.InterfaceName, config.IPAMResult.IP4.IP, sandboxName, routeCommands),
			c.CommandBuilder.IdempotentlySetupBridge(vxlanName, sandboxLinkName, sandboxName, config.BridgeName, config.IPAMResult),
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
		ID:        config.ContainerID,
		MAC:       getHardwareAddressCommand.Result.String(),
		IP:        config.IPAMResult.IP4.IP.IP.String(),
		NetworkID: config.NetworkID,
		HostIP:    config.HostIP,
	}, nil
}
