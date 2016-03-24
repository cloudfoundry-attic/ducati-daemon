package container

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/locks"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
)

type Deletor struct {
	Executor          executor.Executor
	NamedLocker       locks.NamedLocker
	Watcher           watcher.MissWatcher
	SandboxRepository namespace.Repository
	NamespaceOpener   namespace.Opener
}

type DeletorConfig struct {
	InterfaceName   string
	ContainerNSPath string
	SandboxNS       namespace.Namespace
	VxlanDeviceName string
}

func (d *Deletor) Delete(deletorConfig DeletorConfig) error {
	containerNS, err := d.NamespaceOpener.OpenPath(deletorConfig.ContainerNSPath)
	if err != nil {
		return fmt.Errorf("open container netns: %s", err)
	}

	err = d.Executor.Execute(
		commands.All(
			commands.InNamespace{
				Namespace: containerNS,
				Command: commands.DeleteLink{
					LinkName: deletorConfig.InterfaceName,
				},
			},

			commands.CleanupSandbox{
				Namespace:         deletorConfig.SandboxNS,
				SandboxRepository: d.SandboxRepository,
				NamedLocker:       d.NamedLocker,
				Watcher:           d.Watcher,
				VxlanDeviceName:   deletorConfig.VxlanDeviceName,
			},
		),
	)
	if err != nil {
		return err
	}

	return nil
}
