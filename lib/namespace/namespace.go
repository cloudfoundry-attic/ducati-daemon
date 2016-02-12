package namespace

import (
	"os"
	"path/filepath"

	"github.com/appc/cni/pkg/ns"
)

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

func (n *namespace) Execute(callback func(*os.File) error) error {
	return ns.WithNetNSPath(n.path, false, func(f *os.File) error {
		return callback(f)
	})
}

func (n *namespace) Destroy() error {
	return unlinkNetworkNamespace(n.path)
}
