package container

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
)

type Deletor struct {
	Executor executor.Executor
	Locker   commands.Locker
}

type DeletorConfig struct {
	InterfaceName   string
	ContainerNSPath string
	SandboxNSPath   string
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
				Namespace: namespace.NewNamespace(deletorConfig.SandboxNSPath),
				Locker:    d.Locker,
			},
		),
	)
	if err != nil {
		return err
	}

	return nil
}
