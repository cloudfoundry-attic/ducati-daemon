package commands

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/sandbox"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
	"github.com/pivotal-golang/lager"
)

type CleanupSandbox struct {
	SandboxName     string
	Watcher         watcher.MissWatcher
	VxlanDeviceName string
}

func (c CleanupSandbox) Execute(context executor.Context) error {
	logger := context.Logger().Session("cleanup-sandbox", lager.Data{"sandbox-name": c.SandboxName})
	logger.Info("start")
	defer logger.Info("complete")

	sbox, err := context.SandboxRepository().Get(c.SandboxName)
	if err != nil {
		if err == sandbox.NotFoundError {
			logger.Error("get-sandbox-failed", err)
			return nil
		}

		logger.Error("get-sandbox-failed", err)
		return fmt.Errorf("get sandbox: %s", err)
	}
	sbox.Lock()
	defer sbox.Unlock()

	logger.Info("starting-namespace-execute")
	var vethLinkCount = 0
	err = sbox.Namespace().Execute(func(ns *os.File) error {
		var err error
		vethLinkCount, err = context.LinkFactory().VethDeviceCount()
		if err != nil {
			return fmt.Errorf("counting veth devices: %s", err)
		}

		logger.Info("veth-links-remaining", lager.Data{"count": vethLinkCount})

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
		logger.Error("namespace-execute", err)
		return fmt.Errorf("in namespace %s: %s", c.SandboxName, err)
	}
	logger.Info("namespace-execute-complete")

	namespaceRepo := context.SandboxNamespaceRepository()

	if vethLinkCount == 0 {
		logger.Info("stop-miss-monitor")

		err := c.Watcher.StopMonitor(sbox.Namespace())
		if err != nil {
			return fmt.Errorf("watcher stop monitor: %s", err)
		}

		logger.Info("calling-namespace-destroy")

		err = namespaceRepo.Destroy(sbox.Namespace())
		if err != nil {
			return fmt.Errorf("destroying sandbox %s: %s", c.SandboxName, err)
		}

		logger.Info("removing-sandbox")
		context.SandboxRepository().Remove(c.SandboxName)
	}

	return nil
}

func (c CleanupSandbox) String() string {
	return fmt.Sprintf("cleanup-sandbox %s", c.SandboxName)
}
