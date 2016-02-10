package ipam

import "github.com/appc/cni/plugins/ipam/host-local/backend/disk"

type StoreFactory struct{}

func (sf *StoreFactory) Create(path string) (AllocatorStore, error) {
	return disk.New(path)
}
