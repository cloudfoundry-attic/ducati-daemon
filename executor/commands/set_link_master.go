package commands

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type SetLinkMaster struct {
	Master string
	Slave  string
}

func (slm SetLinkMaster) Execute(context executor.Context) error {
	err := context.LinkFactory().SetMaster(slm.Slave, slm.Master)
	if err != nil {
		return fmt.Errorf("set link master: %s", err)
	}

	return nil
}

func (slm SetLinkMaster) String() string {
	return fmt.Sprintf("ip link set %s master %s", slm.Slave, slm.Master)
}
