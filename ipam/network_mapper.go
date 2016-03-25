package ipam

import (
	"crypto/sha1"
	"encoding/binary"
)

//go:generate counterfeiter -o ../fakes/network_mapper.go --fake-name NetworkMapper . NetworkMapper
type NetworkMapper interface {
	GetVNI(networkID string) (int, error)
}

type FixedNetworkMapper struct{}

func (*FixedNetworkMapper) GetVNI(networkID string) (int, error) {
	digest := sha1.Sum([]byte(networkID))
	digest[3] = 0

	vni := binary.LittleEndian.Uint32(digest[:4])

	return int(vni), nil
}
