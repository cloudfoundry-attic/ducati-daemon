package commands

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
)

type StartMonitor struct {
	HostNamespace namespace.Namespace
	Watcher       watcher.MissWatcher
	SandboxName   string
	VxlanLinkName string
}

func (sm StartMonitor) Execute(context executor.Context) error {
	ns, err := context.SandboxRepository().Get(sm.SandboxName)
	if err != nil {
		return fmt.Errorf("getting sandbox namespace: %s", err)
	}

	return sm.HostNamespace.Execute(func(_ *os.File) error {
		err = sm.Watcher.StartMonitor(ns, sm.VxlanLinkName)
		if err != nil {
			return fmt.Errorf("watcher start monitor: %s", err)
		}
		return nil
	})
}

func (sm StartMonitor) String() string {
	return fmt.Sprintf("ip netns exec %s ip monitor neigh", sm.SandboxName)
}
