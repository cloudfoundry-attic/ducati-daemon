package commands

import (
	"fmt"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type SetHardwareAddress struct {
	LinkName        string
	HardwareAddress net.HardwareAddr
}

func (cmd *SetHardwareAddress) Execute(context executor.Context) error {
	err := context.LinkFactory().SetHardwareAddress(cmd.LinkName, cmd.HardwareAddress)
	if err != nil {
		return fmt.Errorf("set hardware address: %s", err)
	}

	return nil
}

func (cmd *SetHardwareAddress) String() string {
	return fmt.Sprintf("ip link set %s address %s", cmd.LinkName, cmd.HardwareAddress)
}
