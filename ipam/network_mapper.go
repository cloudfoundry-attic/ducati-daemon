package ipam

import (
	"crypto/sha1"
	"encoding/hex"
	"math/big"
)

//go:generate counterfeiter -o ../fakes/network_mapper.go --fake-name NetworkMapper . NetworkMapper
type NetworkMapper interface {
	GetVNI(networkID string) (int, error)
}

type FixedNetworkMapper struct{}

func (m *FixedNetworkMapper) GetVNI(networkID string) (int, error) {
	digest := sha1.Sum([]byte(networkID))

	hexSha1 := hex.EncodeToString(digest[:20])[:5]

	vni := new(big.Int)
	vni.SetString(hexSha1, 32)

	return int(vni.Uint64()), nil
}
