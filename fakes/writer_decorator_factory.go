// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/miekg/dns"
)

type WriterDecoratorFactory struct {
	DecorateStub        func(namespace.Namespace) dns.DecorateWriter
	decorateMutex       sync.RWMutex
	decorateArgsForCall []struct {
		arg1 namespace.Namespace
	}
	decorateReturns struct {
		result1 dns.DecorateWriter
	}
}

func (fake *WriterDecoratorFactory) Decorate(arg1 namespace.Namespace) dns.DecorateWriter {
	fake.decorateMutex.Lock()
	fake.decorateArgsForCall = append(fake.decorateArgsForCall, struct {
		arg1 namespace.Namespace
	}{arg1})
	fake.decorateMutex.Unlock()
	if fake.DecorateStub != nil {
		return fake.DecorateStub(arg1)
	} else {
		return fake.decorateReturns.result1
	}
}

func (fake *WriterDecoratorFactory) DecorateCallCount() int {
	fake.decorateMutex.RLock()
	defer fake.decorateMutex.RUnlock()
	return len(fake.decorateArgsForCall)
}

func (fake *WriterDecoratorFactory) DecorateArgsForCall(i int) namespace.Namespace {
	fake.decorateMutex.RLock()
	defer fake.decorateMutex.RUnlock()
	return fake.decorateArgsForCall[i].arg1
}

func (fake *WriterDecoratorFactory) DecorateReturns(result1 dns.DecorateWriter) {
	fake.DecorateStub = nil
	fake.decorateReturns = struct {
		result1 dns.DecorateWriter
	}{result1}
}
