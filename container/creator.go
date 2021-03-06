package container

import (
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"net"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/sandbox"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
)

//go:generate counterfeiter -o ../fakes/command_builder.go --fake-name CommandBuilder . commandBuilder
type commandBuilder interface {
	IdempotentlyCreateSandbox(sandboxName, vxlanName string, vni int, dnsAddress string) executor.Command
	IdempotentlyCreateVxlan(vxlanName string, sandboxName string, sandboxNS namespace.Namespace) executor.Command
	AddRoutes(interfaceName string, ipConfig *types.IPConfig) executor.Command
	SetupVeth(containerNS namespace.Namespace, sandboxLinkName string, containerLinkName string, address net.IPNet, sandboxName string, routeCommand executor.Command) executor.Command
	IdempotentlySetupBridge(vxlanName, sandboxLinkName, bridgeName string, sandboxNS namespace.Namespace, ipamResult *types.Result) executor.Command
}

type sandboxRepository interface {
	Get(sandboxName string) (sandbox.Sandbox, error)
}

type Creator struct {
	Executor        executor.Executor
	SandboxRepo     sandboxRepository
	Watcher         watcher.MissWatcher
	CommandBuilder  commandBuilder
	DNSAddress      string
	HostIP          net.IP
	NamespaceOpener namespace.Opener
}

type CreatorConfig struct {
	NetworkID       string
	App             string
	ContainerNsPath string
	ContainerID     string
	InterfaceName   string
	VNI             int
	IPAMResult      *types.Result
}

func NameSandboxLink(containerID string) string {
	const maxLength = 15
	hash := sha1.Sum([]byte(containerID))
	return string(base32.StdEncoding.EncodeToString(hash[:])[:maxLength])
}

func (c *Creator) Setup(config CreatorConfig) (models.Container, error) {
	vxlanName := fmt.Sprintf("vxlan%d", config.VNI)
	sandboxName := fmt.Sprintf("vni-%d", config.VNI)
	bridgeName := fmt.Sprintf("vxlanbr%d", config.VNI)

	containerNS, err := c.NamespaceOpener.OpenPath(config.ContainerNsPath)
	if err != nil {
		return models.Container{}, fmt.Errorf("open container netns: %s", err)
	}

	sandboxLinkName := NameSandboxLink(config.ContainerID)

	var routeCommands = c.CommandBuilder.AddRoutes(config.InterfaceName, config.IPAMResult.IP4)

	err = c.Executor.Execute(c.CommandBuilder.IdempotentlyCreateSandbox(sandboxName, vxlanName, config.VNI, c.DNSAddress))
	if err != nil {
		return models.Container{}, fmt.Errorf("executing command: create sandbox: %s", err)
	}

	sandbox, err := c.SandboxRepo.Get(sandboxName)
	if err != nil {
		return models.Container{}, fmt.Errorf("get sandbox: %s", err)
	}
	sandbox.Lock()
	defer sandbox.Unlock()

	sandboxNS := sandbox.Namespace()
	err = c.Executor.Execute(
		commands.All(
			c.CommandBuilder.IdempotentlyCreateVxlan(vxlanName, sandboxName, sandboxNS),
			c.CommandBuilder.SetupVeth(containerNS, sandboxLinkName, config.InterfaceName, config.IPAMResult.IP4.IP, sandboxName, routeCommands),
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
		ID:          config.ContainerID,
		MAC:         getHardwareAddressCommand.Result.String(),
		IP:          config.IPAMResult.IP4.IP.IP.String(),
		NetworkID:   config.NetworkID,
		HostIP:      c.HostIP.String(),
		SandboxName: sandboxName,
		App:         config.App,
	}, nil
}
