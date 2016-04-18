package conditions

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type SandboxExists struct {
	Name string
}

func (n SandboxExists) Satisfied(context executor.Context) bool {
	sbox := context.SandboxRepository().Get(n.Name)
	return sbox != nil
}

func (n SandboxExists) String() string {
	return fmt.Sprintf(`check if sandbox "%s" exists`, n.Name)
}
