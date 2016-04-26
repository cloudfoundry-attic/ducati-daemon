package network

import (
	"crypto/sha1"
	"encoding/binary"

	"github.com/cloudfoundry-incubator/ducati-daemon/models"
)

//go:generate counterfeiter -o ../fakes/network_mapper.go --fake-name NetworkMapper . NetworkMapper
type NetworkMapper interface {
	GetVNI(networkID string) (int, error)
	GetNetworkID(netPayload models.NetworkPayload) (string, error)
}

type FixedNetworkMapper struct {
	DefaultNetworkID string
}

func (*FixedNetworkMapper) GetVNI(networkID string) (int, error) {
	digest := sha1.Sum([]byte(networkID))
	digest[3] = 0

	vni := binary.LittleEndian.Uint32(digest[:4])

	return int(vni), nil
}

func (m *FixedNetworkMapper) GetNetworkID(netPayload models.NetworkPayload) (string, error) {
	if netPayload.Properties.SpaceGUID == "" {
		return m.DefaultNetworkID, nil
	}
	return netPayload.Properties.SpaceGUID, nil
}
