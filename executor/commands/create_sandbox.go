package commands

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type CreateSandbox struct {
	Name string
}

func (cn CreateSandbox) Execute(context executor.Context) error {
	_, err := context.SandboxRepository().Create(cn.Name)
	if err != nil {
		return fmt.Errorf("create sandbox: %s", err)
	}

	return nil
}

func (cn CreateSandbox) String() string {
	return fmt.Sprintf("create sandbox %s", cn.Name)
}
