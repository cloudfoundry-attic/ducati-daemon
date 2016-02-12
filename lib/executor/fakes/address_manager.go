// This file was generated by counterfeiter
package fakes

import (
	"net"
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/executor"
	"github.com/vishvananda/netlink"
)

type AddressManager struct {
	AddAddressStub        func(link netlink.Link, address *net.IPNet) error
	addAddressMutex       sync.RWMutex
	addAddressArgsForCall []struct {
		link    netlink.Link
		address *net.IPNet
	}
	addAddressReturns struct {
		result1 error
	}
}

func (fake *AddressManager) AddAddress(link netlink.Link, address *net.IPNet) error {
	fake.addAddressMutex.Lock()
	fake.addAddressArgsForCall = append(fake.addAddressArgsForCall, struct {
		link    netlink.Link
		address *net.IPNet
	}{link, address})
	fake.addAddressMutex.Unlock()
	if fake.AddAddressStub != nil {
		return fake.AddAddressStub(link, address)
	} else {
		return fake.addAddressReturns.result1
	}
}

func (fake *AddressManager) AddAddressCallCount() int {
	fake.addAddressMutex.RLock()
	defer fake.addAddressMutex.RUnlock()
	return len(fake.addAddressArgsForCall)
}

func (fake *AddressManager) AddAddressArgsForCall(i int) (netlink.Link, *net.IPNet) {
	fake.addAddressMutex.RLock()
	defer fake.addAddressMutex.RUnlock()
	return fake.addAddressArgsForCall[i].link, fake.addAddressArgsForCall[i].address
}

func (fake *AddressManager) AddAddressReturns(result1 error) {
	fake.AddAddressStub = nil
	fake.addAddressReturns = struct {
		result1 error
	}{result1}
}

var _ executor.AddressManager = new(AddressManager)
