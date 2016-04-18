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
	Executor             executor.Executor
	NamedLocker          locks.NamedLocker
	Watcher              watcher.MissWatcher
	SandboxNamespaceRepo namespace.Repository
	NamespaceOpener      namespace.Opener
}

func (d *Deletor) Delete(
	interfaceName string,
	containerNSPath string,
	sandboxNS namespace.Namespace,
	vxlanDeviceName string,
) error {
	containerNS, err := d.NamespaceOpener.OpenPath(containerNSPath)
	if err != nil {
		return fmt.Errorf("open container netns: %s", err)
	}

	err = d.Executor.Execute(
		commands.All(
			commands.InNamespace{
				Namespace: containerNS,
				Command: commands.DeleteLink{
					LinkName: interfaceName,
				},
			},

			commands.CleanupSandbox{
				Namespace:         sandboxNS,
				SandboxRepository: d.SandboxNamespaceRepo,
				NamedLocker:       d.NamedLocker,
				Watcher:           d.Watcher,
				VxlanDeviceName:   vxlanDeviceName,
			},
		),
	)
	if err != nil {
		return err
	}

	return nil
}
