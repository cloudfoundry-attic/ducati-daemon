package commands

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/sandbox"
	"github.com/pivotal-golang/lager"
)

type CleanupSandbox struct {
	SandboxName     string
	VxlanDeviceName string
}

func (c CleanupSandbox) Execute(context executor.Context) error {
	logger := context.Logger().Session("cleanup-sandbox", lager.Data{"sandbox-name": c.SandboxName})
	logger.Info("start")
	defer logger.Info("complete")

	sandboxRepo := context.SandboxRepository()
	sbox, err := sandboxRepo.Get(c.SandboxName)
	if err != nil {
		if err == sandbox.NotFoundError {
			return nil
		}

		logger.Error("get-sandbox-failed", err)
		return fmt.Errorf("get sandbox: %s", err)
	}

	// TODO: move into sandbox
	sbox.Lock()
	defer sbox.Unlock()

	vethLinkCount, err := sbox.VethDeviceCount()
	if err != nil {
		return fmt.Errorf("counting veth devices: %s", err)
	}

	logger.Info("veth-links-remaining", lager.Data{"count": vethLinkCount})

	if vethLinkCount == 0 {
		err = sbox.Namespace().Execute(func(*os.File) error {
			err := context.LinkFactory().DeleteLinkByName(c.VxlanDeviceName)
			if err != nil {
				if context.LinkFactory().Exists(c.VxlanDeviceName) {
					return fmt.Errorf("destroying vxlan %s: %s", c.VxlanDeviceName, err)
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("in namespace %s: %s", c.SandboxName, err)
		}

		err = sandboxRepo.Destroy(c.SandboxName)
		switch err {
		case nil:
		case sandbox.AlreadyDestroyedError:
		case sandbox.NotFoundError:
		default:
			return fmt.Errorf("sandbox destroy: %s", err)
		}
	}

	return nil
}

func (c CleanupSandbox) String() string {
	return fmt.Sprintf("cleanup-sandbox %s", c.SandboxName)
}
