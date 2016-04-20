package sandbox

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

var NotFoundError = errors.New("not found")
var AlreadyExistsError = errors.New("already exists")

//go:generate counterfeiter -o ../fakes/sandbox_repository.go --fake-name SandboxRepository . Repository
type Repository interface {
	Create(sandboxName string) (Sandbox, error)
	Get(sandboxName string) (Sandbox, error)
	Remove(sandboxName string)
}

//go:generate counterfeiter -o ../fakes/invoker.go --fake-name Invoker . Invoker
type Invoker interface {
	Invoke(ifrit.Runner) ifrit.Process
}

type InvokeFunc func(ifrit.Runner) ifrit.Process

func (i InvokeFunc) Invoke(r ifrit.Runner) ifrit.Process {
	return i(r)
}

type repository struct {
	logger        lager.Logger
	sandboxes     map[string]*sandbox
	locker        sync.Locker
	namespaceRepo namespace.Repository
	invoker       Invoker
}

func NewRepository(
	logger lager.Logger,
	locker sync.Locker,
	namespaceRepo namespace.Repository,
	invoker Invoker,
) Repository {
	return &repository{
		logger:        logger,
		sandboxes:     map[string]*sandbox{},
		locker:        locker,
		namespaceRepo: namespaceRepo,
		invoker:       invoker,
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

	sandbox := New(r.logger, ns, r.invoker)
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
