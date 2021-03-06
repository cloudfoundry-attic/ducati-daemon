// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
)

type AddController struct {
	AddStub        func(models.CNIAddPayload) (*types.Result, error)
	addMutex       sync.RWMutex
	addArgsForCall []struct {
		arg1 models.CNIAddPayload
	}
	addReturns struct {
		result1 *types.Result
		result2 error
	}
}

func (fake *AddController) Add(arg1 models.CNIAddPayload) (*types.Result, error) {
	fake.addMutex.Lock()
	fake.addArgsForCall = append(fake.addArgsForCall, struct {
		arg1 models.CNIAddPayload
	}{arg1})
	fake.addMutex.Unlock()
	if fake.AddStub != nil {
		return fake.AddStub(arg1)
	} else {
		return fake.addReturns.result1, fake.addReturns.result2
	}
}

func (fake *AddController) AddCallCount() int {
	fake.addMutex.RLock()
	defer fake.addMutex.RUnlock()
	return len(fake.addArgsForCall)
}

func (fake *AddController) AddArgsForCall(i int) models.CNIAddPayload {
	fake.addMutex.RLock()
	defer fake.addMutex.RUnlock()
	return fake.addArgsForCall[i].arg1
}

func (fake *AddController) AddReturns(result1 *types.Result, result2 error) {
	fake.AddStub = nil
	fake.addReturns = struct {
		result1 *types.Result
		result2 error
	}{result1, result2}
}
