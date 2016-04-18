package commands

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type CreateSandboxNamespace struct {
	Name string
}

func (cn CreateSandboxNamespace) Execute(context executor.Context) error {
	_, err := context.SandboxNamespaceRepository().Create(cn.Name)
	if err != nil {
		return fmt.Errorf("create namespace: %s", err)
	}

	return nil
}

func (cn CreateSandboxNamespace) String() string {
	return fmt.Sprintf("ip netns add %s", cn.Name)
}
