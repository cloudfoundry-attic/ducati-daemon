// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/appc/cni/pkg/types"
)

type ConfigFactory struct {
	CreateStub        func(networkID string) (types.IPConfig, error)
	createMutex       sync.RWMutex
	createArgsForCall []struct {
		networkID string
	}
	createReturns struct {
		result1 types.IPConfig
		result2 error
	}
}

func (fake *ConfigFactory) Create(networkID string) (types.IPConfig, error) {
	fake.createMutex.Lock()
	fake.createArgsForCall = append(fake.createArgsForCall, struct {
		networkID string
	}{networkID})
	fake.createMutex.Unlock()
	if fake.CreateStub != nil {
		return fake.CreateStub(networkID)
	} else {
		return fake.createReturns.result1, fake.createReturns.result2
	}
}

func (fake *ConfigFactory) CreateCallCount() int {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	return len(fake.createArgsForCall)
}

func (fake *ConfigFactory) CreateArgsForCall(i int) string {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	return fake.createArgsForCall[i].networkID
}

func (fake *ConfigFactory) CreateReturns(result1 types.IPConfig, result2 error) {
	fake.CreateStub = nil
	fake.createReturns = struct {
		result1 types.IPConfig
		result2 error
	}{result1, result2}
}
