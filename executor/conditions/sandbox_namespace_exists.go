package conditions

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type SandboxNamespaceExists struct {
	Name string
}

func (n SandboxNamespaceExists) Satisfied(context executor.Context) bool {
	_, err := context.SandboxRepository().Get(n.Name)
	return err == nil
}

func (n SandboxNamespaceExists) String() string {
	return fmt.Sprintf(`check if sandbox "%s" exists`, n.Name)
}
