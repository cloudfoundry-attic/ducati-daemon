package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
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
	Load(string) error
	ForEach(SandboxCallback) error
}

//go:generate counterfeiter -o ../fakes/invoker.go --fake-name Invoker . Invoker
type Invoker interface {
	Invoke(ifrit.Runner) ifrit.Process
}

//go:generate counterfeiter -o ../fakes/sandbox_callback.go --fake-name SandboxCallback . SandboxCallback
type SandboxCallback interface {
	Callback(ns namespace.Namespace) error
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
	linkFactory   linkFactory
	watcher       watcher.MissWatcher
}

func NewRepository(
	logger lager.Logger,
	locker sync.Locker,
	namespaceRepo namespace.Repository,
	invoker Invoker,
	linkFactory linkFactory,
	watcher watcher.MissWatcher,
) Repository {
	return &repository{
		logger:        logger,
		sandboxes:     map[string]*sandbox{},
		locker:        locker,
		namespaceRepo: namespaceRepo,
		invoker:       invoker,
		linkFactory:   linkFactory,
		watcher:       watcher,
	}
}

func (r *repository) Load(sandboxRepoDir string) error {
	r.locker.Lock()
	defer r.locker.Unlock()

	err := filepath.Walk(sandboxRepoDir, func(filePath string, f os.FileInfo, err error) error {
		// skip root dir
		if sandboxRepoDir == filePath {
			return nil
		}

		sandboxName := path.Base(filePath)

		ns, err := r.namespaceRepo.Get(sandboxName)
		if err != nil {
			return fmt.Errorf("loading sandbox repo: %s", err)
		}
		sandbox := New(r.logger, ns, r.invoker, r.linkFactory)
		r.sandboxes[sandboxName] = sandbox

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) ForEach(s SandboxCallback) error {
	r.locker.Lock()
	defer r.locker.Unlock()

	for _, sbox := range r.sandboxes {
		err := s.Callback(sbox.Namespace())
		if err != nil {
			return fmt.Errorf("callback: %s", err)
		}
	}
	return nil
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

	sandbox := New(r.logger, ns, r.invoker, r.linkFactory, r.watcher)
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
