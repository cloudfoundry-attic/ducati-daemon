package sandbox

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

//go:generate counterfeiter -o ../fakes/runner.go --fake-name Runner . runner
type runner interface {
	ifrit.Runner
}

//go:generate counterfeiter -o ../fakes/process.go --fake-name Process . process
type process interface {
	ifrit.Process
}

//go:generate counterfeiter -o ../fakes/sandbox.go --fake-name Sandbox . Sandbox
type Sandbox interface {
	sync.Locker
	Namespace() namespace.Namespace
	LaunchDNS(ifrit.Runner) error
}

type sandbox struct {
	sync.Mutex
	namespace namespace.Namespace
	invoker   Invoker
	logger    lager.Logger
}

func New(
	logger lager.Logger,
	namespace namespace.Namespace,
	invoker Invoker,
) *sandbox {
	logger = logger.Session("sandbox", lager.Data{"namespace": namespace.Name()})

	return &sandbox{
		namespace: namespace,
		invoker:   invoker,
		logger:    logger,
	}
}

func (s *sandbox) Namespace() namespace.Namespace {
	return s.namespace
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
