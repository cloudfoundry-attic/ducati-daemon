package sandbox

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

const LOOPBACK_DEVICE_NAME = "lo"

//go:generate counterfeiter -o ../fakes/runner.go --fake-name Runner . runner
type runner interface {
	ifrit.Runner
}

//go:generate counterfeiter -o ../fakes/process.go --fake-name Process . process
type process interface {
	ifrit.Process
}

type linkFactory interface {
	SetUp(name string) error
}

//go:generate counterfeiter -o ../fakes/sandbox.go --fake-name Sandbox . Sandbox
type Sandbox interface {
	sync.Locker
	Setup() error
	Namespace() namespace.Namespace
	LaunchDNS(ifrit.Runner) error
}

type sandbox struct {
	sync.Mutex
	namespace   namespace.Namespace
	invoker     Invoker
	logger      lager.Logger
	linkFactory linkFactory
}

func New(
	logger lager.Logger,
	namespace namespace.Namespace,
	invoker Invoker,
	linkFactory linkFactory,
) *sandbox {
	logger = logger.Session("sandbox", lager.Data{"namespace": namespace.Name()})

	return &sandbox{
		logger:      logger,
		namespace:   namespace,
		invoker:     invoker,
		linkFactory: linkFactory,
	}
}

func (s *sandbox) Namespace() namespace.Namespace {
	return s.namespace
}

func (s *sandbox) Setup() error {
	err := s.namespace.Execute(func(*os.File) error {
		err := s.linkFactory.SetUp(LOOPBACK_DEVICE_NAME)
		if err != nil {
			return fmt.Errorf("set link up: %s", err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("setup failed: %s", err)
	}

	return nil
}

func (s *sandbox) LaunchDNS(dns ifrit.Runner) error {
	s.logger.Info("launch-dns")
	process := s.invoker.Invoke(dns)

	select {
	case err := <-process.Wait():
		if err == nil {
			err = errors.New("unexpected server exit")
		}
		return fmt.Errorf("launch dns: %s", err)
	default:
		return nil
	}
}
