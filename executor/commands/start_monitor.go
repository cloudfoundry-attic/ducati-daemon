package commands

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
)

type StartMonitor struct {
	Watcher   watcher.MissWatcher
	Namespace namespace.Namespace
}

func (sm StartMonitor) Execute(context Context) error {
	err := sm.Watcher.StartMonitor(sm.Namespace)
	if err != nil {
		return fmt.Errorf("watcher start monitor: %s", err)
	}
	return nil
}

func (sm StartMonitor) String() string {
	return fmt.Sprintf("ip netns exec %s ip monitor neigh", sm.Namespace.Name())
}
