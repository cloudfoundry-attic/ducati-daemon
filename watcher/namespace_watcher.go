package watcher

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
)

type NamespaceWatcher struct {
	MissWatcher MissWatcher
	Namespace   namespace.Namespace
	DeviceName  string
}

func (nw *NamespaceWatcher) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	err := nw.MissWatcher.StartMonitor(nw.Namespace, nw.DeviceName)
	if err != nil {
		return fmt.Errorf("start monitor: %s", err)
	}

	close(ready)
	<-signals

	err = nw.MissWatcher.StopMonitor(nw.Namespace)
	if err != nil {
		return fmt.Errorf("stop monitor: %s", err)
	}

	return nil
}
