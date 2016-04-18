package sandbox

import "sync"

type Repository interface {
	Get(sandboxName string) *Sandbox
	Put(sandboxName string, sandbox *Sandbox)
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

func (r *repository) Put(sandboxName string, sandbox *Sandbox) {
	r.locker.Lock()
	r.sandboxes[sandboxName] = sandbox
	r.locker.Unlock()
	return
}

func (r *repository) Remove(sandboxName string) {
	r.locker.Lock()
	delete(r.sandboxes, sandboxName)
	r.locker.Unlock()
	return
}
