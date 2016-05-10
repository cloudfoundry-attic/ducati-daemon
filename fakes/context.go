// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/pivotal-golang/lager"
)

type Context struct {
	LoggerStub        func() lager.Logger
	loggerMutex       sync.RWMutex
	loggerArgsForCall []struct{}
	loggerReturns     struct {
		result1 lager.Logger
	}
	AddressManagerStub        func() executor.AddressManager
	addressManagerMutex       sync.RWMutex
	addressManagerArgsForCall []struct{}
	addressManagerReturns     struct {
		result1 executor.AddressManager
	}
	LinkFactoryStub        func() executor.LinkFactory
	linkFactoryMutex       sync.RWMutex
	linkFactoryArgsForCall []struct{}
	linkFactoryReturns     struct {
		result1 executor.LinkFactory
	}
	RouteManagerStub        func() executor.RouteManager
	routeManagerMutex       sync.RWMutex
	routeManagerArgsForCall []struct{}
	routeManagerReturns     struct {
		result1 executor.RouteManager
	}
	SandboxNamespaceRepositoryStub        func() namespace.Repository
	sandboxNamespaceRepositoryMutex       sync.RWMutex
	sandboxNamespaceRepositoryArgsForCall []struct{}
	sandboxNamespaceRepositoryReturns     struct {
		result1 namespace.Repository
	}
	SandboxRepositoryStub        func() executor.SandboxRepository
	sandboxRepositoryMutex       sync.RWMutex
	sandboxRepositoryArgsForCall []struct{}
	sandboxRepositoryReturns     struct {
		result1 executor.SandboxRepository
	}
	ListenerFactoryStub        func() executor.ListenerFactory
	listenerFactoryMutex       sync.RWMutex
	listenerFactoryArgsForCall []struct{}
	listenerFactoryReturns     struct {
		result1 executor.ListenerFactory
	}
	DNSServerFactoryStub        func() executor.DNSServerFactory
	dNSServerFactoryMutex       sync.RWMutex
	dNSServerFactoryArgsForCall []struct{}
	dNSServerFactoryReturns     struct {
		result1 executor.DNSServerFactory
	}
}

func (fake *Context) Logger() lager.Logger {
	fake.loggerMutex.Lock()
	fake.loggerArgsForCall = append(fake.loggerArgsForCall, struct{}{})
	fake.loggerMutex.Unlock()
	if fake.LoggerStub != nil {
		return fake.LoggerStub()
	} else {
		return fake.loggerReturns.result1
	}
}

func (fake *Context) LoggerCallCount() int {
	fake.loggerMutex.RLock()
	defer fake.loggerMutex.RUnlock()
	return len(fake.loggerArgsForCall)
}

func (fake *Context) LoggerReturns(result1 lager.Logger) {
	fake.LoggerStub = nil
	fake.loggerReturns = struct {
		result1 lager.Logger
	}{result1}
}

func (fake *Context) AddressManager() executor.AddressManager {
	fake.addressManagerMutex.Lock()
	fake.addressManagerArgsForCall = append(fake.addressManagerArgsForCall, struct{}{})
	fake.addressManagerMutex.Unlock()
	if fake.AddressManagerStub != nil {
		return fake.AddressManagerStub()
	} else {
		return fake.addressManagerReturns.result1
	}
}

func (fake *Context) AddressManagerCallCount() int {
	fake.addressManagerMutex.RLock()
	defer fake.addressManagerMutex.RUnlock()
	return len(fake.addressManagerArgsForCall)
}

func (fake *Context) AddressManagerReturns(result1 executor.AddressManager) {
	fake.AddressManagerStub = nil
	fake.addressManagerReturns = struct {
		result1 executor.AddressManager
	}{result1}
}

func (fake *Context) LinkFactory() executor.LinkFactory {
	fake.linkFactoryMutex.Lock()
	fake.linkFactoryArgsForCall = append(fake.linkFactoryArgsForCall, struct{}{})
	fake.linkFactoryMutex.Unlock()
	if fake.LinkFactoryStub != nil {
		return fake.LinkFactoryStub()
	} else {
		return fake.linkFactoryReturns.result1
	}
}

func (fake *Context) LinkFactoryCallCount() int {
	fake.linkFactoryMutex.RLock()
	defer fake.linkFactoryMutex.RUnlock()
	return len(fake.linkFactoryArgsForCall)
}

func (fake *Context) LinkFactoryReturns(result1 executor.LinkFactory) {
	fake.LinkFactoryStub = nil
	fake.linkFactoryReturns = struct {
		result1 executor.LinkFactory
	}{result1}
}

func (fake *Context) RouteManager() executor.RouteManager {
	fake.routeManagerMutex.Lock()
	fake.routeManagerArgsForCall = append(fake.routeManagerArgsForCall, struct{}{})
	fake.routeManagerMutex.Unlock()
	if fake.RouteManagerStub != nil {
		return fake.RouteManagerStub()
	} else {
		return fake.routeManagerReturns.result1
	}
}

func (fake *Context) RouteManagerCallCount() int {
	fake.routeManagerMutex.RLock()
	defer fake.routeManagerMutex.RUnlock()
	return len(fake.routeManagerArgsForCall)
}

func (fake *Context) RouteManagerReturns(result1 executor.RouteManager) {
	fake.RouteManagerStub = nil
	fake.routeManagerReturns = struct {
		result1 executor.RouteManager
	}{result1}
}

func (fake *Context) SandboxNamespaceRepository() namespace.Repository {
	fake.sandboxNamespaceRepositoryMutex.Lock()
	fake.sandboxNamespaceRepositoryArgsForCall = append(fake.sandboxNamespaceRepositoryArgsForCall, struct{}{})
	fake.sandboxNamespaceRepositoryMutex.Unlock()
	if fake.SandboxNamespaceRepositoryStub != nil {
		return fake.SandboxNamespaceRepositoryStub()
	} else {
		return fake.sandboxNamespaceRepositoryReturns.result1
	}
}

func (fake *Context) SandboxNamespaceRepositoryCallCount() int {
	fake.sandboxNamespaceRepositoryMutex.RLock()
	defer fake.sandboxNamespaceRepositoryMutex.RUnlock()
	return len(fake.sandboxNamespaceRepositoryArgsForCall)
}

func (fake *Context) SandboxNamespaceRepositoryReturns(result1 namespace.Repository) {
	fake.SandboxNamespaceRepositoryStub = nil
	fake.sandboxNamespaceRepositoryReturns = struct {
		result1 namespace.Repository
	}{result1}
}

func (fake *Context) SandboxRepository() executor.SandboxRepository {
	fake.sandboxRepositoryMutex.Lock()
	fake.sandboxRepositoryArgsForCall = append(fake.sandboxRepositoryArgsForCall, struct{}{})
	fake.sandboxRepositoryMutex.Unlock()
	if fake.SandboxRepositoryStub != nil {
		return fake.SandboxRepositoryStub()
	} else {
		return fake.sandboxRepositoryReturns.result1
	}
}

func (fake *Context) SandboxRepositoryCallCount() int {
	fake.sandboxRepositoryMutex.RLock()
	defer fake.sandboxRepositoryMutex.RUnlock()
	return len(fake.sandboxRepositoryArgsForCall)
}

func (fake *Context) SandboxRepositoryReturns(result1 executor.SandboxRepository) {
	fake.SandboxRepositoryStub = nil
	fake.sandboxRepositoryReturns = struct {
		result1 executor.SandboxRepository
	}{result1}
}

func (fake *Context) ListenerFactory() executor.ListenerFactory {
	fake.listenerFactoryMutex.Lock()
	fake.listenerFactoryArgsForCall = append(fake.listenerFactoryArgsForCall, struct{}{})
	fake.listenerFactoryMutex.Unlock()
	if fake.ListenerFactoryStub != nil {
		return fake.ListenerFactoryStub()
	} else {
		return fake.listenerFactoryReturns.result1
	}
}

func (fake *Context) ListenerFactoryCallCount() int {
	fake.listenerFactoryMutex.RLock()
	defer fake.listenerFactoryMutex.RUnlock()
	return len(fake.listenerFactoryArgsForCall)
}

func (fake *Context) ListenerFactoryReturns(result1 executor.ListenerFactory) {
	fake.ListenerFactoryStub = nil
	fake.listenerFactoryReturns = struct {
		result1 executor.ListenerFactory
	}{result1}
}

func (fake *Context) DNSServerFactory() executor.DNSServerFactory {
	fake.dNSServerFactoryMutex.Lock()
	fake.dNSServerFactoryArgsForCall = append(fake.dNSServerFactoryArgsForCall, struct{}{})
	fake.dNSServerFactoryMutex.Unlock()
	if fake.DNSServerFactoryStub != nil {
		return fake.DNSServerFactoryStub()
	} else {
		return fake.dNSServerFactoryReturns.result1
	}
}

func (fake *Context) DNSServerFactoryCallCount() int {
	fake.dNSServerFactoryMutex.RLock()
	defer fake.dNSServerFactoryMutex.RUnlock()
	return len(fake.dNSServerFactoryArgsForCall)
}

func (fake *Context) DNSServerFactoryReturns(result1 executor.DNSServerFactory) {
	fake.DNSServerFactoryStub = nil
	fake.dNSServerFactoryReturns = struct {
		result1 executor.DNSServerFactory
	}{result1}
}

var _ executor.Context = new(Context)
