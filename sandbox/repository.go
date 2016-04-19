package sandbox

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
)

var NotFoundError = errors.New("not found")
var AlreadyExistsError = errors.New("already exists")

//go:generate counterfeiter -o ../fakes/sandbox_repository.go --fake-name SandboxRepository . Repository
type Repository interface {
	Create(sandboxName string) (Sandbox, error)
	Get(sandboxName string) (Sandbox, error)
	Remove(sandboxName string)
}

type repository struct {
	sandboxes     map[string]*sandbox
	locker        sync.Locker
	namespaceRepo namespace.Repository
}

func NewRepository(locker sync.Locker, namespaceRepo namespace.Repository) Repository {
	return &repository{
		sandboxes:     map[string]*sandbox{},
		locker:        locker,
		namespaceRepo: namespaceRepo,
	}
}

func (r *repository) Create(sandboxName string) (Sandbox, error) {
	r.locker.Lock()
	defer r.locker.Unlock()

	if _, exists := r.sandboxes[sandboxName]; exists {
		return nil, fmt.Errorf("sandbox %q already exists", sandboxName)
	}

	ns, err := r.namespaceRepo.Create(sandboxName)
	if err != nil {
		return nil, fmt.Errorf("create namespace: %s", err)
	}

	sandbox := New(ns)
	r.sandboxes[sandboxName] = sandbox

	return sandbox, nil
}

func (r *repository) Get(sandboxName string) (Sandbox, error) {
	r.locker.Lock()
	sbox, exists := r.sandboxes[sandboxName]
	r.locker.Unlock()

	if !exists {
		return nil, NotFoundError
	}
	return sbox, nil
}

func (r *repository) Remove(sandboxName string) {
	r.locker.Lock()
	delete(r.sandboxes, sandboxName)
	r.locker.Unlock()
	return
}
