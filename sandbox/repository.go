package sandbox

import (
	"fmt"
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
)

//go:generate counterfeiter -o ../fakes/sandbox_repository.go --fake-name SandboxRepository . Repository
type Repository interface {
	Create(sandboxName string) (*Sandbox, error)
	Get(sandboxName string) *Sandbox
	Remove(sandboxName string)
}

type repository struct {
	sandboxes     map[string]*Sandbox
	locker        sync.Locker
	namespaceRepo namespace.Repository
}

func NewRepository(locker sync.Locker, namespaceRepo namespace.Repository) Repository {
	return &repository{
		sandboxes:     map[string]*Sandbox{},
		locker:        locker,
		namespaceRepo: namespaceRepo,
	}
}

func (r *repository) Create(sandboxName string) (*Sandbox, error) {
	r.locker.Lock()
	defer r.locker.Unlock()

	if _, exists := r.sandboxes[sandboxName]; exists {
		return nil, fmt.Errorf("sandbox %q already exists", sandboxName)
	}

	ns, err := r.namespaceRepo.Create(sandboxName)
	if err != nil {
		return nil, fmt.Errorf("create namespace: %s", err)
	}

	sandbox := &Sandbox{
		Namespace: ns,
	}
	r.sandboxes[sandboxName] = sandbox

	return sandbox, nil
}

func (r *repository) Get(sandboxName string) *Sandbox {
	r.locker.Lock()
	sbox := r.sandboxes[sandboxName]
	r.locker.Unlock()
	return sbox
}


func (r *repository) Remove(sandboxName string) {
	r.locker.Lock()
	delete(r.sandboxes, sandboxName)
	r.locker.Unlock()
	return
}
