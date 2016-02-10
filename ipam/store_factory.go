package ipam

import "sync"

type StoreFactory struct{}

func (sf *StoreFactory) Create(networkID string) (AllocatorStore, error) {
	return NewStore(&sync.Mutex{}), nil
}
