package conditions

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

//go:generate counterfeiter --fake-name LinkFinder . LinkFinder
type LinkFinder interface {
	Exists(name string) bool
}

type LinkExists struct {
	LinkFinder LinkFinder
	Name       string
}

func (l LinkExists) Satisfied(context executor.Context) bool {
	return l.LinkFinder.Exists(l.Name)
}

func (l LinkExists) String() string {
	return fmt.Sprintf(`check if link "%s" exists`, l.Name)
}
