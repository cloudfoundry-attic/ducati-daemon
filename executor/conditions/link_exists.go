package conditions

import "fmt"

//go:generate counterfeiter --fake-name LinkFinder . LinkFinder
type LinkFinder interface {
	Exists(name string) bool
}

type LinkExists struct {
	LinkFinder LinkFinder
	Name       string
}

func (l LinkExists) Satisfied(context Context) bool {
	return l.LinkFinder.Exists(l.Name)
}

func (l LinkExists) String() string {
	return fmt.Sprintf(`check if link "%s" exists`, l.Name)
}
