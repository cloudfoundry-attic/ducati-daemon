package commands

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
)

//go:generate counterfeiter --fake-name Locker . Locker
type Locker interface {
	Lock(string)
	Unlock(string)
}

type CleanupSandbox struct {
	Namespace       namespace.Namespace
	Locker          Locker
	Watcher         watcher.MissWatcher
	VxlanDeviceName string
}

func (c CleanupSandbox) Execute(context executor.Context) error {
	sandboxName := c.Namespace.Name()

	c.Locker.Lock(sandboxName)
	defer c.Locker.Unlock(sandboxName)

	var vethLinkCount = 0
	err := c.Namespace.Execute(func(ns *os.File) error {
		var err error
		vethLinkCount, err = context.LinkFactory().VethDeviceCount()
		if err != nil {
			return fmt.Errorf("counting veth devices: %s", err)
		}

		if vethLinkCount == 0 {
			err = context.LinkFactory().DeleteLinkByName(c.VxlanDeviceName)
			if err != nil {
				return fmt.Errorf("destroying vxlan %s: %s", c.VxlanDeviceName, err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("in namespace %s: %s", sandboxName, err)
	}

	if vethLinkCount == 0 {
		err := c.Watcher.StopMonitor(c.Namespace)
		if err != nil {
			return fmt.Errorf("watcher stop monitor: %s", err)
		}

		err = c.Namespace.Destroy()
		if err != nil {
			return fmt.Errorf("destroying sandbox %s: %s", sandboxName, err)
		}
	}

	return nil
}

func (c CleanupSandbox) String() string {
	return fmt.Sprintf("cleanup-sandbox %s", c.Namespace.Name())
}
