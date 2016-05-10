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

type Repository struct {
	Logger        lager.Logger
	Locker        sync.Locker
	NamespaceRepo namespace.Repository
	Invoker       Invoker
	LinkFactory   linkFactory
	Watcher       watcher.MissWatcher

	Sandboxes map[string]Sandbox
}

func (r *Repository) Load(sandboxRepoDir string) error {
	r.Locker.Lock()
	defer r.Locker.Unlock()

	err := filepath.Walk(sandboxRepoDir, func(filePath string, f os.FileInfo, err error) error {
		// skip root dir
		if sandboxRepoDir == filePath {
			return nil
		}

		sandboxName := path.Base(filePath)

		ns, err := r.NamespaceRepo.Get(sandboxName)
		if err != nil {
			return fmt.Errorf("loading sandbox repo: %s", err)
		}

		sandbox := New(r.Logger, ns, r.Invoker, r.LinkFactory, r.Watcher)
		r.Sandboxes[sandboxName] = sandbox

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) ForEach(s SandboxCallback) error {
	r.Locker.Lock()
	defer r.Locker.Unlock()

	for _, sbox := range r.Sandboxes {
		err := s.Callback(sbox.Namespace())
		if err != nil {
			return fmt.Errorf("callback: %s", err)
		}
	}
	return nil
}

func (r *Repository) Create(sandboxName string) (Sandbox, error) {
	logger := r.Logger.Session("create", lager.Data{"name": sandboxName})
	logger.Info("starting")
	defer logger.Info("complete")

	r.Locker.Lock()
	defer r.Locker.Unlock()

	if _, exists := r.Sandboxes[sandboxName]; exists {
		return nil, AlreadyExistsError
	}

	ns, err := r.NamespaceRepo.Create(sandboxName)
	if err != nil {
		return nil, fmt.Errorf("create namespace: %s", err)
	}

	sandbox := New(r.Logger, ns, r.Invoker, r.LinkFactory, r.Watcher)
	r.Sandboxes[sandboxName] = sandbox

	return sandbox, nil
}

func (r *Repository) Get(sandboxName string) (Sandbox, error) {
	r.Locker.Lock()
	defer r.Locker.Unlock()

	return r.get(sandboxName)
}

func (r *Repository) get(sandboxName string) (Sandbox, error) {
	sbox, exists := r.Sandboxes[sandboxName]
	if !exists {
		return nil, NotFoundError
	}

	return sbox, nil
}

func (r *Repository) Destroy(sandboxName string) error {
	logger := r.Logger.Session("destroy", lager.Data{"name": sandboxName})
	logger.Info("starting")
	defer logger.Info("complete")

	r.Locker.Lock()
	defer r.Locker.Unlock()

	sbox, err := r.get(sandboxName)
	if err != nil {
		return err
	}

	err = sbox.Teardown()
	if err != nil {
		return fmt.Errorf("teardown: %s", err)
	}

	err = r.NamespaceRepo.Destroy(sbox.Namespace())
	if err != nil {
		return fmt.Errorf("namespace destroy: %s", err)
	}

	delete(r.Sandboxes, sandboxName)

	return nil
}
