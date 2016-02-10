// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/pivotal-golang/lager"
)

type Logger struct {
	DebugStub        func(action string, data ...lager.Data)
	debugMutex       sync.RWMutex
	debugArgsForCall []struct {
		action string
		data   []lager.Data
	}
	ErrorStub        func(action string, err error, data ...lager.Data)
	errorMutex       sync.RWMutex
	errorArgsForCall []struct {
		action string
		err    error
		data   []lager.Data
	}
}

func (fake *Logger) Debug(action string, data ...lager.Data) {
	fake.debugMutex.Lock()
	fake.debugArgsForCall = append(fake.debugArgsForCall, struct {
		action string
		data   []lager.Data
	}{action, data})
	fake.debugMutex.Unlock()
	if fake.DebugStub != nil {
		fake.DebugStub(action, data...)
	}
}

func (fake *Logger) DebugCallCount() int {
	fake.debugMutex.RLock()
	defer fake.debugMutex.RUnlock()
	return len(fake.debugArgsForCall)
}

func (fake *Logger) DebugArgsForCall(i int) (string, []lager.Data) {
	fake.debugMutex.RLock()
	defer fake.debugMutex.RUnlock()
	return fake.debugArgsForCall[i].action, fake.debugArgsForCall[i].data
}

func (fake *Logger) Error(action string, err error, data ...lager.Data) {
	fake.errorMutex.Lock()
	fake.errorArgsForCall = append(fake.errorArgsForCall, struct {
		action string
		err    error
		data   []lager.Data
	}{action, err, data})
	fake.errorMutex.Unlock()
	if fake.ErrorStub != nil {
		fake.ErrorStub(action, err, data...)
	}
}

func (fake *Logger) ErrorCallCount() int {
	fake.errorMutex.RLock()
	defer fake.errorMutex.RUnlock()
	return len(fake.errorArgsForCall)
}

func (fake *Logger) ErrorArgsForCall(i int) (string, error, []lager.Data) {
	fake.errorMutex.RLock()
	defer fake.errorMutex.RUnlock()
	return fake.errorArgsForCall[i].action, fake.errorArgsForCall[i].err, fake.errorArgsForCall[i].data
}

var _ handlers.Logger = new(Logger)
