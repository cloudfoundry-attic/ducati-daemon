package commands

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

//go:generate counterfeiter --fake-name Namespace . Namespace
type Namespace interface {
	Execute(func(*os.File) error) error
	Path() string
}

type InNamespace struct {
	Namespace Namespace
	Command   executor.Command
}

func (i InNamespace) Execute(context executor.Context) error {
	err := i.Namespace.Execute(func(_ *os.File) error {
		return i.Command.Execute(context)
	})
	if err != nil {
		return fmt.Errorf("execute in namespace: %s", err)
	}

	return nil
}

func (i InNamespace) String() string {
	return fmt.Sprintf("ip netns exec %s %s", i.Namespace.Path(), i.Command)
}
