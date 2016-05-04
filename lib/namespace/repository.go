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

	"github.com/cloudfoundry-incubator/ducati-daemon/ossupport"
	"github.com/pivotal-golang/lager"
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
	logger       lager.Logger
	root         string
	threadLocker ossupport.OSThreadLocker
}

func NewRepository(logger lager.Logger, root string, threadLocker ossupport.OSThreadLocker) (Repository, error) {
	err := os.MkdirAll(root, 0755)
	if err != nil {
		return nil, err
	}
	return &repository{
		logger:       logger,
		root:         root,
		threadLocker: threadLocker,
	}, nil
}

func (r *repository) Get(name string) (Namespace, error) {
	logger := r.logger.Session("get")

	file, err := r.open(name)
	if err != nil {
		logger.Error("open-failed", err)
		return nil, err
	}

	ns := &Netns{Logger: logger, File: file, ThreadLocker: r.threadLocker}

	logger.Info("complete", lager.Data{"namespace": ns})

	return ns, nil
}

func (r *repository) PathOf(path string) string {
	return filepath.Join(r.root, path)
}

func (r *repository) Create(name string) (Namespace, error) {
	logger := r.logger.Session("create")

	tempName := fmt.Sprintf("ns-%.08x", random())
	err := exec.Command("ip", "netns", "add", tempName).Run()
	if err != nil {
		logger.Error("ip-netns-add-failed", err)
		return nil, err
	}

	netnsPath := filepath.Join("/var/run/netns", tempName)
	bindMountedFile, err := bindMountFile(netnsPath, r.PathOf(name))
	if err != nil {
		logger.Error("bind-mount-failed", err)
		return nil, err
	}

	err = unlinkNetworkNamespace(netnsPath)
	if err != nil {
		logger.Error("unlink-failed", err)
		return nil, err
	}

	ns := &Netns{Logger: logger, File: bindMountedFile, ThreadLocker: r.threadLocker}

	logger.Info("created", lager.Data{"namespace": ns})

	return ns, nil
}

func (r *repository) Destroy(namespace Namespace) error {
	logger := r.logger.Session("destroy")

	ns, ok := namespace.(*Netns)
	if !ok {
		logger.Error("not-a-netns", nil)
		return errors.New("namespace is not a Netns")
	}

	logger.Info("repo-data", lager.Data{"name": ns.Name(), "root": r.root})
	if !strings.HasPrefix(ns.Name(), r.root) {
		logger.Error("outside-of-repo", nil, lager.Data{"name": namespace.Name()})
		return fmt.Errorf("namespace outside of repository: %s", ns.Name())
	}

	logger.Info("destroying", lager.Data{"namespace": namespace})

	err := unlinkNetworkNamespace(ns.File.Name())
	if err != nil {
		logger.Error("unlink-failed", err)
	}

	return err
}

func (r *repository) open(name string) (*os.File, error) {
	return os.Open(filepath.Join(r.root, name))
}

func random() uint32 {
	return rand.Uint32()
}
