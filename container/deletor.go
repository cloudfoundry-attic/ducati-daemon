package container

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/locks"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
)

type Deletor struct {
	Executor    executor.Executor
	NamedLocker locks.NamedLocker
	Watcher     watcher.MissWatcher
}

type DeletorConfig struct {
	InterfaceName   string
	ContainerNSPath string
	SandboxNSPath   string
	VxlanDeviceName string
}

func (d *Deletor) Delete(deletorConfig DeletorConfig) error {
	err := d.Executor.Execute(
		commands.All(
			commands.InNamespace{
				Namespace: namespace.NewNamespace(deletorConfig.ContainerNSPath),
				Command: commands.DeleteLink{
					LinkName: deletorConfig.InterfaceName,
				},
			},

			commands.CleanupSandbox{
				Namespace:       namespace.NewNamespace(deletorConfig.SandboxNSPath),
				NamedLocker:     d.NamedLocker,
				Watcher:         d.Watcher,
				VxlanDeviceName: deletorConfig.VxlanDeviceName,
			},
		),
	)
	if err != nil {
		return err
	}

	return nil
}
