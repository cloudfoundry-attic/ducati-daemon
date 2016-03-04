package namespace

import (
	"os"
	"path/filepath"
)

//go:generate counterfeiter -o ../../fakes/namespace.go --fake-name Namespace . Namespace
type Namespace interface {
	Destroy() error
	Execute(func(*os.File) error) error
	Name() string
	Open() (*os.File, error)
	Path() string
}

type namespace struct {
	path string
}

func NewNamespace(path string) Namespace {
	return &namespace{
		path: path,
	}
}

func (n *namespace) Name() string {
	return filepath.Base(n.path)
}

func (n *namespace) Open() (*os.File, error) {
	return os.Open(n.Path())
}

func (n *namespace) Path() string {
	return n.path
}

func (n *namespace) Destroy() error {
	return unlinkNetworkNamespace(n.path)
}
