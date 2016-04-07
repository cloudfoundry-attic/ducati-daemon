package ipam

import (
	"net"
	"sync"
)

type inMemoryStore struct {
	allocated map[string]string
	locker    sync.Locker
}

func NewStore(locker sync.Locker) *inMemoryStore {
	return &inMemoryStore{
		allocated: map[string]string{},
		locker:    locker,
	}
}

func (s *inMemoryStore) Reserve(id string, ip net.IP) (bool, error) {
	s.locker.Lock()
	defer s.locker.Unlock()

	key := ip.String()
	_, ok := s.allocated[key]
	if ok {
		return false, nil
	}

	s.allocated[key] = id
	return true, nil
}

func (s *inMemoryStore) ReleaseByID(id string) error {
	s.locker.Lock()
	defer s.locker.Unlock()

	for k, v := range s.allocated {
		if v == id {
			delete(s.allocated, k)
		}
	}

	return nil
}

func (s *inMemoryStore) Contains(id string) bool {
	for _, idInStore := range s.allocated {
		if idInStore == id {
			return true
		}
	}
	return false
}
