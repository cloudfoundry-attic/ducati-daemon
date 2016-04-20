package sandbox

import (
	"errors"
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/tedsuo/ifrit"
)

//go:generate counterfeiter -o ../fakes/runner.go --fake-name Runner . runner
type runner interface {
	ifrit.Runner
}

//go:generate counterfeiter -o ../fakes/invoker.go --fake-name Invoker . invoker
type invoker interface {
	Invoke(ifrit.Runner) ifrit.Process
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

func New(namespace namespace.Namespace) *sandbox {
	return &sandbox{
		namespace: namespace,
	}
}

type sandbox struct {
	sync.Mutex
	namespace namespace.Namespace
}

func (s *sandbox) Namespace() namespace.Namespace {
	return s.namespace
}

func (s *sandbox) LaunchDNS(ifrit.Runner) error {
	return errors.New("not implemented")
}
