package container

import (
	"fmt"
	"net"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/locks"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
)

//go:generate counterfeiter -o ../fakes/command_builder.go --fake-name CommandBuilder . commandBuilder
type commandBuilder interface {
	IdempotentlyCreateSandbox(sandboxName, vxlanDeviceName string) executor.Command
	IdempotentlyCreateVxlan(vxlanName string, vni int, sandboxName string, sandboxNS namespace.Namespace) executor.Command
	AddRoutes(interfaceName string, ipConfig *types.IPConfig) executor.Command
	SetupVeth(containerNS namespace.Namespace, sandboxLinkName string, containerLinkName string, address net.IPNet, sandboxNS namespace.Namespace, routeCommand executor.Command) executor.Command
	IdempotentlySetupBridge(vxlanName, sandboxLinkName, bridgeName string, sandboxNS namespace.Namespace, ipamResult *types.Result) executor.Command
}

type Creator struct {
	Executor        executor.Executor
	SandboxRepo     namespace.Repository
	NamedLocker     locks.NamedLocker
	Watcher         watcher.MissWatcher
	CommandBuilder  commandBuilder
	HostIP          net.IP
	NamespaceOpener namespace.Opener
}

type CreatorConfig struct {
	NetworkID       string
	ContainerNsPath string
	ContainerID     string
	InterfaceName   string
	VNI             int
	IPAMResult      *types.Result
}

func (c *Creator) Setup(config CreatorConfig) (models.Container, error) {
	vxlanName := fmt.Sprintf("vxlan%d", config.VNI)
	sandboxName := fmt.Sprintf("vni-%d", config.VNI)
	bridgeName := fmt.Sprintf("vxlanbr%d", config.VNI)

	containerNS, err := c.NamespaceOpener.OpenPath(config.ContainerNsPath)
	if err != nil {
		return models.Container{}, fmt.Errorf("open container netns: %s", err)
	}

	sandboxLinkName := config.ContainerID
	if len(sandboxLinkName) > 15 {
		sandboxLinkName = sandboxLinkName[:15]
	}

	var routeCommands = c.CommandBuilder.AddRoutes(config.InterfaceName, config.IPAMResult.IP4)

	c.NamedLocker.Lock(sandboxName)
	defer c.NamedLocker.Unlock(sandboxName)

	err = c.Executor.Execute(c.CommandBuilder.IdempotentlyCreateSandbox(sandboxName, vxlanName))
	if err != nil {
		return models.Container{}, fmt.Errorf("executing command: create sandbox: %s", err)
	}

	sandboxNS, err := c.SandboxRepo.Get(sandboxName)
	if err != nil {
		return models.Container{}, fmt.Errorf("get sandbox: %s", err)
	}
	err = c.Executor.Execute(
		commands.All(
			c.CommandBuilder.IdempotentlyCreateVxlan(vxlanName, config.VNI, sandboxName, sandboxNS),
			c.CommandBuilder.SetupVeth(containerNS, sandboxLinkName, config.InterfaceName, config.IPAMResult.IP4.IP, sandboxNS, routeCommands),
			c.CommandBuilder.IdempotentlySetupBridge(vxlanName, sandboxLinkName, bridgeName, sandboxNS, config.IPAMResult),
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
		HostIP:    c.HostIP.String(),
	}, nil
}
