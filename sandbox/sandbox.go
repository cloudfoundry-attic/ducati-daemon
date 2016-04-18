package sandbox

import (
	"os"

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

type Sandbox struct {
	Invoker          invoker
	Namespace        namespace.Namespace
	NamespaceWatcher runner
}

func (s *Sandbox) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	process := s.Invoker.Invoke(s.NamespaceWatcher)
	close(ready)

	signal := <-signals
	process.Signal(signal)

	return nil
}
