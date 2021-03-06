package cni

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/network"
	"github.com/cloudfoundry-incubator/ducati-daemon/ossupport"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
)

//go:generate counterfeiter -o ../fakes/deletor.go --fake-name Deletor . deletor
type deletor interface {
	Delete(interfaceName string, containerNSPath string, sandboxName string, vxlanDeviceName string) error
}

type repository interface {
	Get(string) (namespace.Namespace, error)
}

type DelController struct {
	Datastore      store.Store
	Deletor        deletor
	IPAllocator    ipam.IPAllocator
	NetworkMapper  network.NetworkMapper
	OSThreadLocker ossupport.OSThreadLocker
}

func (c *DelController) Del(payload models.CNIDelPayload) error {
	dbRecord, err := c.Datastore.Get(payload.ContainerID)
	if err != nil {
		return fmt.Errorf("datastore get: %s", err)
	}

	vni, err := c.NetworkMapper.GetVNI(dbRecord.NetworkID)
	if err != nil {
		return fmt.Errorf("get vni: %s", err)
	}

	sandboxName := fmt.Sprintf("vni-%d", vni)
	vxlanDeviceName := fmt.Sprintf("vxlan%d", vni)

	err = c.Deletor.Delete(payload.InterfaceName, payload.ContainerNamespace, sandboxName, vxlanDeviceName)
	if err != nil {
		return fmt.Errorf("deletor: %s", err)
	}

	err = c.Datastore.Delete(payload.ContainerID)
	if err != nil {
		return fmt.Errorf("datastore delete: %s", err)
	}

	err = c.IPAllocator.ReleaseIP(dbRecord.NetworkID, payload.ContainerID)
	if err != nil {
		return fmt.Errorf("release ip: %s", err)
	}

	return nil
}
