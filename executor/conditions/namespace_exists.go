package conditions

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
)

type NamespaceExists struct {
	Name       string
	Repository namespace.Repository
}

func (n NamespaceExists) Satisfied(context Context) bool {
	_, err := n.Repository.Get(n.Name)
	return err == nil
}

func (n NamespaceExists) String() string {
	return fmt.Sprintf(`check if network namespace "%s" exists`, n.Repository.PathOf(n.Name))
}
