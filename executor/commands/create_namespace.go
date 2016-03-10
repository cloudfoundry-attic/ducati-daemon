package commands

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
)

type CreateNamespace struct {
	Name       string
	Repository namespace.Repository
}

func (cn CreateNamespace) Execute(context executor.Context) error {
	_, err := cn.Repository.Create(cn.Name)
	if err != nil {
		return fmt.Errorf("create namespace: %s", err)
	}

	return nil
}

func (cn CreateNamespace) String() string {
	return fmt.Sprintf("ip netns add %s", cn.Name)
}
