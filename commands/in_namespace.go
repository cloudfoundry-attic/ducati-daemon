package commands

import (
	"fmt"
	"os"
)

//go:generate counterfeiter --fake-name Namespace . Namespace
type Namespace interface {
	Execute(func(*os.File) error) error
	Path() string
}

type InNamespace struct {
	Namespace Namespace
	Command   Command
}

func (i InNamespace) Execute(context Context) error {
	err := i.Namespace.Execute(func(_ *os.File) error {
		return i.Command.Execute(context)
	})
	if err != nil {
		return fmt.Errorf("execute in namespace: %s", err)
	}

	return nil
}
