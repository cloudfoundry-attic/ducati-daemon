// This file was generated by counterfeiter
package fakes

import (
	"os"
	"sync"
)

type Runner struct {
	RunStub        func(signals <-chan os.Signal, ready chan<- struct{}) error
	runMutex       sync.RWMutex
	runArgsForCall []struct {
		signals <-chan os.Signal
		ready   chan<- struct{}
	}
	runReturns struct {
		result1 error
	}
}

func (fake *Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	fake.runMutex.Lock()
	fake.runArgsForCall = append(fake.runArgsForCall, struct {
		signals <-chan os.Signal
		ready   chan<- struct{}
	}{signals, ready})
	fake.runMutex.Unlock()
	if fake.RunStub != nil {
		return fake.RunStub(signals, ready)
	} else {
		return fake.runReturns.result1
	}
}

func (fake *Runner) RunCallCount() int {
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	return len(fake.runArgsForCall)
}

func (fake *Runner) RunArgsForCall(i int) (<-chan os.Signal, chan<- struct{}) {
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	return fake.runArgsForCall[i].signals, fake.runArgsForCall[i].ready
}

func (fake *Runner) RunReturns(result1 error) {
	fake.RunStub = nil
	fake.runReturns = struct {
		result1 error
	}{result1}
}