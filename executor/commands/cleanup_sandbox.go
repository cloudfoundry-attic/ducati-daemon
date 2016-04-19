package commands

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
)

type CleanupSandbox struct {
	SandboxName     string
	Watcher         watcher.MissWatcher
	VxlanDeviceName string
}

func (c CleanupSandbox) Execute(context executor.Context) error {
	sbox, err := context.SandboxRepository().Get(c.SandboxName)
	if err != nil {
		return fmt.Errorf("get sandbox: %s", err)
	}
	sbox.Lock()
	defer sbox.Unlock()

	var vethLinkCount = 0
	err = sbox.Namespace().Execute(func(ns *os.File) error {
		var err error
		vethLinkCount, err = context.LinkFactory().VethDeviceCount()
		if err != nil {
			return fmt.Errorf("counting veth devices: %s", err)
		}

		if vethLinkCount == 0 {
			err = context.LinkFactory().DeleteLinkByName(c.VxlanDeviceName)
			if err != nil {
				if context.LinkFactory().Exists(c.VxlanDeviceName) {
					return fmt.Errorf("destroying vxlan %s: %s", c.VxlanDeviceName, err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("in namespace %s: %s", c.SandboxName, err)
	}

	namespaceRepo := context.SandboxNamespaceRepository()

	if vethLinkCount == 0 {
		err := c.Watcher.StopMonitor(sbox.Namespace())
		if err != nil {
			return fmt.Errorf("watcher stop monitor: %s", err)
		}

		err = namespaceRepo.Destroy(sbox.Namespace())
		if err != nil {
			return fmt.Errorf("destroying sandbox %s: %s", c.SandboxName, err)
		}
	}

	context.SandboxRepository().Remove(c.SandboxName)

	return nil
}

func (c CleanupSandbox) String() string {
	return fmt.Sprintf("cleanup-sandbox %s", c.SandboxName)
}
