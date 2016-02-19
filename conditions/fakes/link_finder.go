// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/conditions"
)

type LinkFinder struct {
	ExistsStub        func(name string) bool
	existsMutex       sync.RWMutex
	existsArgsForCall []struct {
		name string
	}
	existsReturns struct {
		result1 bool
	}
}

func (fake *LinkFinder) Exists(name string) bool {
	fake.existsMutex.Lock()
	fake.existsArgsForCall = append(fake.existsArgsForCall, struct {
		name string
	}{name})
	fake.existsMutex.Unlock()
	if fake.ExistsStub != nil {
		return fake.ExistsStub(name)
	} else {
		return fake.existsReturns.result1
	}
}

func (fake *LinkFinder) ExistsCallCount() int {
	fake.existsMutex.RLock()
	defer fake.existsMutex.RUnlock()
	return len(fake.existsArgsForCall)
}

func (fake *LinkFinder) ExistsArgsForCall(i int) string {
	fake.existsMutex.RLock()
	defer fake.existsMutex.RUnlock()
	return fake.existsArgsForCall[i].name
}

func (fake *LinkFinder) ExistsReturns(result1 bool) {
	fake.ExistsStub = nil
	fake.existsReturns = struct {
		result1 bool
	}{result1}
}

var _ conditions.LinkFinder = new(LinkFinder)
