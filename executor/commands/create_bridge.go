package commands

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type CreateBridge struct {
	Name string
}

func (cb CreateBridge) Execute(context executor.Context) error {
	err := context.LinkFactory().CreateBridge(cb.Name)
	if err != nil {
		return fmt.Errorf("create bridge: %s", err)
	}

	return nil
}

func (cb CreateBridge) String() string {
	return fmt.Sprintf("ip link add dev %s type bridge", cb.Name)
}
