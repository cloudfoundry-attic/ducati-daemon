package namespace

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

//go:generate counterfeiter --fake-name Repository -o ../../fakes/repository.go . Repository
type Repository interface {
	Get(name string) (Namespace, error)
	Create(name string) (Namespace, error)
	PathOf(path string) string
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

type repository struct {
	root string
}

func NewRepository(root string) (Repository, error) {
	err := os.MkdirAll(root, 0755)
	if err != nil {
		return nil, err
	}
	return &repository{
		root: root,
	}, nil
}

func (r *repository) Get(name string) (Namespace, error) {
	file, err := r.open(name)
	if err != nil {
		return nil, err
	}
	file.Close()

	return NewNamespace(file.Name()), nil
}

func (r *repository) Create(name string) (Namespace, error) {
	file, err := r.create(name)
	if err != nil {
		return nil, err
	}
	file.Close()

	tempName := fmt.Sprintf("ns-%.08x", random())
	err = exec.Command("ip", "netns", "add", tempName).Run()
	if err != nil {
		os.Remove(file.Name())
		return nil, err
	}

	netnsPath := filepath.Join("/var/run/netns", tempName)
	err = bindMountFile(netnsPath, file.Name())
	if err != nil {
		return nil, err
	}

	err = unlinkNetworkNamespace(netnsPath)
	if err != nil {
		return nil, err
	}

	return NewNamespace(file.Name()), nil
}

func (r *repository) PathOf(path string) string {
	return filepath.Join(r.root, path)
}

func (r *repository) open(name string) (*os.File, error) {
	return os.Open(filepath.Join(r.root, name))
}

func (r *repository) create(name string) (*os.File, error) {
	return os.OpenFile(filepath.Join(r.root, name), os.O_CREATE|os.O_EXCL, 0644)
}

func random() uint32 {
	return rand.Uint32()
}
