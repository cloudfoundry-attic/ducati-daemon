package namespace

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:generate counterfeiter --fake-name Repository -o ../../fakes/repository.go . Repository
type Repository interface {
	Get(name string) (Namespace, error)
	Create(name string) (Namespace, error)
	Destroy(ns Namespace) error
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

	return &Netns{file}, nil
}

func (r *repository) PathOf(path string) string {
	return filepath.Join(r.root, path)
}

func (r *repository) Create(name string) (Namespace, error) {
	tempName := fmt.Sprintf("ns-%.08x", random())
	err := exec.Command("ip", "netns", "add", tempName).Run()
	if err != nil {
		return nil, err
	}

	netnsPath := filepath.Join("/var/run/netns", tempName)
	bindMountedFile, err := bindMountFile(netnsPath, r.PathOf(name))
	if err != nil {
		return nil, err
	}

	err = unlinkNetworkNamespace(netnsPath)
	if err != nil {
		return nil, err
	}

	return &Netns{File: bindMountedFile}, nil
}

func (r *repository) Destroy(namespace Namespace) error {
	ns, ok := namespace.(*Netns)
	if !ok {
		return errors.New("namespace is not a Netns")
	}

	if !strings.HasPrefix(ns.Name(), r.root) {
		return fmt.Errorf("namespace outside of repository: %s", ns.Name())
	}

	return unlinkNetworkNamespace(ns.File.Name())
}

func (r *repository) open(name string) (*os.File, error) {
	return os.Open(filepath.Join(r.root, name))
}

func random() uint32 {
	return rand.Uint32()
}
