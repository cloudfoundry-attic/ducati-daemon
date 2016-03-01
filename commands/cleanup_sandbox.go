package commands

import (
	"fmt"
	"os"
)

//go:generate counterfeiter --fake-name Locker . Locker
type Locker interface {
	Lock(string)
	Unlock(string)
}

//go:generate counterfeiter --fake-name CleanableNamespace . cleanableNamespace
type cleanableNamespace interface {
	Destroy() error
	Name() string
	Namespace
}

type VethDeviceCounter interface {
	VethDeviceCount() (int, error)
}
type CleanupSandbox struct {
	Namespace cleanableNamespace
	Locker    Locker
}

func (c CleanupSandbox) Execute(context Context) error {
	sandboxName := c.Namespace.Name()

	c.Locker.Lock(sandboxName)
	defer c.Locker.Unlock(sandboxName)

	var vethLinkCount = 0
	err := c.Namespace.Execute(func(ns *os.File) error {
		var err error
		vethLinkCount, err = context.VethDeviceCounter().VethDeviceCount()
		if err != nil {
			return fmt.Errorf("counting veth devices: %s", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("in namespace %s: %s", sandboxName, err)
	}

	if vethLinkCount == 0 {
		err := c.Namespace.Destroy()
		if err != nil {
			return fmt.Errorf("destroying sandbox %s: %s", sandboxName, err)
		}
	}

	return nil
}
