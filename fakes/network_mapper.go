// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
)

type NetworkMapper struct {
	GetVNIStub        func(networkID string) (int, error)
	getVNIMutex       sync.RWMutex
	getVNIArgsForCall []struct {
		networkID string
	}
	getVNIReturns struct {
		result1 int
		result2 error
	}
}

func (fake *NetworkMapper) GetVNI(networkID string) (int, error) {
	fake.getVNIMutex.Lock()
	fake.getVNIArgsForCall = append(fake.getVNIArgsForCall, struct {
		networkID string
	}{networkID})
	fake.getVNIMutex.Unlock()
	if fake.GetVNIStub != nil {
		return fake.GetVNIStub(networkID)
	} else {
		return fake.getVNIReturns.result1, fake.getVNIReturns.result2
	}
}

func (fake *NetworkMapper) GetVNICallCount() int {
	fake.getVNIMutex.RLock()
	defer fake.getVNIMutex.RUnlock()
	return len(fake.getVNIArgsForCall)
}

func (fake *NetworkMapper) GetVNIArgsForCall(i int) string {
	fake.getVNIMutex.RLock()
	defer fake.getVNIMutex.RUnlock()
	return fake.getVNIArgsForCall[i].networkID
}

func (fake *NetworkMapper) GetVNIReturns(result1 int, result2 error) {
	fake.GetVNIStub = nil
	fake.getVNIReturns = struct {
		result1 int
		result2 error
	}{result1, result2}
}

var _ ipam.NetworkMapper = new(NetworkMapper)