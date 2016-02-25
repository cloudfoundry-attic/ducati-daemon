package conditions

import "github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"

type NamespaceExists struct {
	Name       string
	Repository namespace.Repository
}

func (n NamespaceExists) Satisfied(_ interface{}) bool {
	_, err := n.Repository.Get(n.Name)
	return err == nil
}
