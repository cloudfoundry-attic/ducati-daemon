package sandbox

import (
	"fmt"
	"sync"
)

type Repository interface {
	Get(sandboxName string) *Sandbox
	Put(sandboxName string, sandbox *Sandbox) error
	Remove(sandboxName string)
}

type repository struct {
	sandboxes map[string]*Sandbox
	locker    sync.Locker
}

func NewRepository(locker sync.Locker) Repository {
	return &repository{
		sandboxes: map[string]*Sandbox{},
		locker:    locker,
	}
}

func (r *repository) Get(sandboxName string) *Sandbox {
	r.locker.Lock()
	sbox := r.sandboxes[sandboxName]
	r.locker.Unlock()
	return sbox
}

func (r *repository) Put(sandboxName string, sandbox *Sandbox) error {
	r.locker.Lock()
	defer r.locker.Unlock()

	if _, exists := r.sandboxes[sandboxName]; exists {
		return fmt.Errorf("sandbox %q already exists", sandboxName)
	}

	r.sandboxes[sandboxName] = sandbox

	return nil
}

func (r *repository) Remove(sandboxName string) {
	r.locker.Lock()
	delete(r.sandboxes, sandboxName)
	r.locker.Unlock()
	return
}
