package commands

import "os"

//go:generate counterfeiter --fake-name Namespace . Namespace
type Namespace interface {
	Execute(func(*os.File) error) error
}

type InNamespace struct {
	Namespace Namespace
	Command   Command
}

func (i InNamespace) Execute(context Context) error {
	return i.Namespace.Execute(func(_ *os.File) error {
		return i.Command.Execute(context)
	})
}
