package cni

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/ossupport"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
)

//go:generate counterfeiter -o ../fakes/deletor.go --fake-name Deletor . deletor
type deletor interface {
	Delete(deletorConfig container.DeletorConfig) error
}

type repository interface {
	Get(string) (namespace.Namespace, error)
}

type DelController struct {
	Datastore      store.Store
	Deletor        deletor
	SandboxRepo    repository
	NetworkMapper  ipam.NetworkMapper
	OSThreadLocker ossupport.OSThreadLocker
}

func (c *DelController) Del(payload models.CNIDelPayload) error {
	c.OSThreadLocker.LockOSThread()
	defer c.OSThreadLocker.UnlockOSThread()

	dbRecord, err := c.Datastore.Get(payload.ContainerID)
	if err != nil {
		return fmt.Errorf("datastore get: %s", err)
	}

	vni, err := c.NetworkMapper.GetVNI(dbRecord.NetworkID)
	if err != nil {
		return fmt.Errorf("get vni: %s", err)
	}

	sandboxName := fmt.Sprintf("vni-%d", vni)
	sandboxNS, err := c.SandboxRepo.Get(sandboxName)
	if err != nil {
		return fmt.Errorf("sandbox get: %s", err)
	}

	deletorConfig := container.DeletorConfig{
		InterfaceName:   payload.InterfaceName,
		SandboxNS:       sandboxNS,
		ContainerNSPath: payload.ContainerNamespace,
		VxlanDeviceName: fmt.Sprintf("vxlan%d", vni),
	}

	err = c.Deletor.Delete(deletorConfig)
	if err != nil {
		return fmt.Errorf("deletor: %s", err)
	}

	err = c.Datastore.Delete(payload.ContainerID)
	if err != nil {
		return fmt.Errorf("datastore delete: %s", err)
	}

	return nil
}
