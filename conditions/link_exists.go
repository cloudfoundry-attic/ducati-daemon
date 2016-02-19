package conditions

//go:generate counterfeiter --fake-name LinkFinder . LinkFinder
type LinkFinder interface {
	Exists(name string) bool
}

type LinkExists struct {
	LinkFinder LinkFinder
	Name       string
}

func (l LinkExists) Satisfied(_ interface{}) bool {
	return l.LinkFinder.Exists(l.Name)
}
