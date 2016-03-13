package ipam

//go:generate counterfeiter -o ../fakes/network_mapper.go --fake-name NetworkMapper . NetworkMapper
type NetworkMapper interface {
	GetVNI(networkID string) (int, error)
}

type FixedNetworkMapper struct {
	VNI int
}

func (m *FixedNetworkMapper) GetVNI(networkID string) (int, error) {
	return m.VNI, nil
}
