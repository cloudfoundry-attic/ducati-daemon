package commands

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
)

type InNamespace struct {
	Namespace namespace.Namespace
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
	return fmt.Sprintf("ip netns exec %s %s", i.Namespace.Name(), i.Command)
}
