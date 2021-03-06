package sandbox

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

const LOOPBACK_DEVICE_NAME = "lo"

var AlreadyDestroyedError = fmt.Errorf("sandbox was already destroyed")

//go:generate counterfeiter -o ../fakes/runner.go --fake-name Runner . runner
type runner interface {
	ifrit.Runner
}

//go:generate counterfeiter -o ../fakes/process.go --fake-name Process . process
type process interface {
	ifrit.Process
}

type LinkFactory interface {
	SetUp(name string) error
	VethDeviceCount() (int, error)
}

//go:generate counterfeiter -o ../fakes/sandbox.go --fake-name Sandbox . Sandbox
type Sandbox interface {
	sync.Locker

	Setup() error
	Teardown() error
	Namespace() namespace.Namespace
	LaunchDNS(ifrit.Runner) error
	VethDeviceCount() (int, error)
}

type NetworkSandbox struct {
	sync.Mutex
	namespace   namespace.Namespace
	invoker     Invoker
	logger      lager.Logger
	linkFactory LinkFactory
	watcher     watcher.MissWatcher

	dnsProcess ifrit.Process
	destroyed  bool
}

func New(
	logger lager.Logger,
	namespace namespace.Namespace,
	invoker Invoker,
	linkFactory LinkFactory,
	watcher watcher.MissWatcher,
) Sandbox {
	logger = logger.Session("network-sandbox", lager.Data{"namespace": namespace.Name()})

	return &NetworkSandbox{
		logger:      logger,
		namespace:   namespace,
		invoker:     invoker,
		linkFactory: linkFactory,
		watcher:     watcher,
	}
}

func (s *NetworkSandbox) Namespace() namespace.Namespace {
	return s.namespace
}

func (s *NetworkSandbox) Setup() error {
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

func (s *NetworkSandbox) LaunchDNS(dns ifrit.Runner) error {
	s.logger.Info("launch-dns")
	s.dnsProcess = s.invoker.Invoke(dns)

	select {
	case err := <-s.dnsProcess.Wait():
		if err == nil {
			err = errors.New("unexpected server exit")
		}
		return fmt.Errorf("launch dns: %s", err)
	default:
		return nil
	}
}

func (s *NetworkSandbox) VethDeviceCount() (int, error) {
	var count int
	var err error
	nserr := s.namespace.Execute(func(*os.File) error {
		count, err = s.linkFactory.VethDeviceCount()
		return nil
	})
	if nserr != nil {
		return 0, fmt.Errorf("namespace execute: %s", nserr)
	}

	if err != nil {
		return 0, fmt.Errorf("veth device count: %s", err)
	}

	return count, nil
}

func (s *NetworkSandbox) Teardown() error {
	if s.destroyed {
		return AlreadyDestroyedError
	}

	err := s.watcher.StopMonitor(s.namespace)
	if err != nil {
		return fmt.Errorf("stop monitor: %s", err)
	}

	if s.dnsProcess != nil {
		s.dnsProcess.Signal(os.Kill)
	}

	s.destroyed = true

	return nil
}
