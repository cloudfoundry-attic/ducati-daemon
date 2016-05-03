// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/miekg/dns"
	"github.com/pivotal-golang/lager"
)

type WriterDecoratorFactory struct {
	DecorateStub        func(lager.Logger, namespace.Namespace) dns.DecorateWriter
	decorateMutex       sync.RWMutex
	decorateArgsForCall []struct {
		arg1 lager.Logger
		arg2 namespace.Namespace
	}
	decorateReturns struct {
		result1 dns.DecorateWriter
	}
}

func (fake *WriterDecoratorFactory) Decorate(arg1 lager.Logger, arg2 namespace.Namespace) dns.DecorateWriter {
	fake.decorateMutex.Lock()
	fake.decorateArgsForCall = append(fake.decorateArgsForCall, struct {
		arg1 lager.Logger
		arg2 namespace.Namespace
	}{arg1, arg2})
	fake.decorateMutex.Unlock()
	if fake.DecorateStub != nil {
		return fake.DecorateStub(arg1, arg2)
	} else {
		return fake.decorateReturns.result1
	}
}

func (fake *WriterDecoratorFactory) DecorateCallCount() int {
	fake.decorateMutex.RLock()
	defer fake.decorateMutex.RUnlock()
	return len(fake.decorateArgsForCall)
}

func (fake *WriterDecoratorFactory) DecorateArgsForCall(i int) (lager.Logger, namespace.Namespace) {
	fake.decorateMutex.RLock()
	defer fake.decorateMutex.RUnlock()
	return fake.decorateArgsForCall[i].arg1, fake.decorateArgsForCall[i].arg2
}

func (fake *WriterDecoratorFactory) DecorateReturns(result1 dns.DecorateWriter) {
	fake.DecorateStub = nil
	fake.decorateReturns = struct {
		result1 dns.DecorateWriter
	}{result1}
}
