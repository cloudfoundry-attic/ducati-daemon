package conditions

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/sandbox"
)

type SandboxExists struct {
	Name string
}

func (n SandboxExists) Satisfied(context executor.Context) (bool, error) {
	_, err := context.SandboxRepository().Get(n.Name)
	if err != nil {
		if err == sandbox.NotFoundError {
			return false, nil
		}
		return false, fmt.Errorf("sandbox get: %s", err)
	}
	return true, nil
}

func (n SandboxExists) String() string {
	return fmt.Sprintf(`check if sandbox "%s" exists`, n.Name)
}
