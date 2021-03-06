// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
)

type Resolver struct {
	ResolveMissesStub        func(misses <-chan watcher.Neighbor, knownNeighbors chan<- watcher.Neighbor)
	resolveMissesMutex       sync.RWMutex
	resolveMissesArgsForCall []struct {
		misses         <-chan watcher.Neighbor
		knownNeighbors chan<- watcher.Neighbor
	}
}

func (fake *Resolver) ResolveMisses(misses <-chan watcher.Neighbor, knownNeighbors chan<- watcher.Neighbor) {
	fake.resolveMissesMutex.Lock()
	fake.resolveMissesArgsForCall = append(fake.resolveMissesArgsForCall, struct {
		misses         <-chan watcher.Neighbor
		knownNeighbors chan<- watcher.Neighbor
	}{misses, knownNeighbors})
	fake.resolveMissesMutex.Unlock()
	if fake.ResolveMissesStub != nil {
		fake.ResolveMissesStub(misses, knownNeighbors)
	}
}

func (fake *Resolver) ResolveMissesCallCount() int {
	fake.resolveMissesMutex.RLock()
	defer fake.resolveMissesMutex.RUnlock()
	return len(fake.resolveMissesArgsForCall)
}

func (fake *Resolver) ResolveMissesArgsForCall(i int) (<-chan watcher.Neighbor, chan<- watcher.Neighbor) {
	fake.resolveMissesMutex.RLock()
	defer fake.resolveMissesMutex.RUnlock()
	return fake.resolveMissesArgsForCall[i].misses, fake.resolveMissesArgsForCall[i].knownNeighbors
}
