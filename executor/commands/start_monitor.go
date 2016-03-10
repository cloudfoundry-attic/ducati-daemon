package commands

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
)

type StartMonitor struct {
	Watcher     watcher.MissWatcher
	SandboxName string
}

func (sm StartMonitor) Execute(context executor.Context) error {
	ns, err := context.SandboxRepository().Get(sm.SandboxName)
	if err != nil {
		return fmt.Errorf("getting sandbox namespace: %s", err)
	}

	err = sm.Watcher.StartMonitor(ns)
	if err != nil {
		return fmt.Errorf("watcher start monitor: %s", err)
	}
	return nil
}

func (sm StartMonitor) String() string {
	return fmt.Sprintf("ip netns exec %s ip monitor neigh", sm.SandboxName)
}
