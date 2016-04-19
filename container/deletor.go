package container

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
)

type Deletor struct {
	Executor        executor.Executor
	Watcher         watcher.MissWatcher
	NamespaceOpener namespace.Opener
}

func (d *Deletor) Delete(
	interfaceName string,
	containerNSPath string,
	sandboxName string,
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
				SandboxName:     sandboxName,
				Watcher:         d.Watcher,
				VxlanDeviceName: vxlanDeviceName,
			},
		),
	)
	if err != nil {
		return err
	}

	return nil
}
