package cni

import (
	"fmt"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/network"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
)

type AddController struct {
	IPAllocator   ipam.IPAllocator
	NetworkMapper network.NetworkMapper
	Creator       creator
	Datastore     store.Store
}

//go:generate counterfeiter -o ../fakes/creator.go --fake-name Creator . creator
type creator interface {
	Setup(container.CreatorConfig) (models.Container, error)
}

func (c *AddController) Add(payload models.CNIAddPayload) (*types.Result, error) {
	networkID, err := c.NetworkMapper.GetNetworkID(payload.Network)
	if err != nil {
		return nil, fmt.Errorf("get network id: %s", err)
	}

	vni, err := c.NetworkMapper.GetVNI(networkID)
	if err != nil {
		return nil, fmt.Errorf("get vni: %s", err)
	}

	ipamResult, err := c.IPAllocator.AllocateIP(networkID, payload.ContainerID)
	if err != nil {
		return nil, err
	}

	containerConfig := container.CreatorConfig{
		NetworkID:       networkID,
		App:             payload.Network.Properties.AppID,
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
