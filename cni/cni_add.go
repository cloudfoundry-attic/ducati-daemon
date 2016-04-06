package cni

import (
	"fmt"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/ossupport"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
)

type AddController struct {
	IPAllocator    ipam.IPAllocator
	NetworkMapper  ipam.NetworkMapper
	Creator        creator
	Datastore      store.Store
	OSThreadLocker ossupport.OSThreadLocker
}

//go:generate counterfeiter -o ../fakes/creator.go --fake-name Creator . creator
type creator interface {
	Setup(container.CreatorConfig) (models.Container, error)
}

func (c *AddController) Add(payload models.CNIAddPayload) (*types.Result, error) {
	c.OSThreadLocker.LockOSThread()
	defer c.OSThreadLocker.UnlockOSThread()

	vni, err := c.NetworkMapper.GetVNI(payload.Network.ID)
	if err != nil {
		return nil, fmt.Errorf("get vni: %s", err)
	}

	ipamResult, err := c.IPAllocator.AllocateIP(payload.Network.ID, payload.ContainerID)
	if err != nil {
		return nil, err
	}

	containerConfig := container.CreatorConfig{
		NetworkID:       payload.Network.ID,
		ContainerNsPath: payload.ContainerNamespace,
		ContainerID:     payload.ContainerID,
		InterfaceName:   payload.InterfaceName,
		VNI:             vni,
		IPAMResult:      ipamResult,
	}

	container, err := c.Creator.Setup(containerConfig)
	if err != nil {
		return nil, fmt.Errorf("container setup: %s", err)
	}

	err = c.Datastore.Create(container)
	if err != nil {
		return nil, fmt.Errorf("datastore create: %s", err)
	}

	return ipamResult, nil
}
