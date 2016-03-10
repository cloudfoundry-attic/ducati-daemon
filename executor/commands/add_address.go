package commands

import (
	"fmt"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type AddAddress struct {
	InterfaceName string
	Address       net.IPNet
}

func (aa AddAddress) Execute(context executor.Context) error {
	err := context.AddressManager().AddAddress(aa.InterfaceName, &aa.Address)
	if err != nil {
		return fmt.Errorf("add address: %s", err)
	}

	return nil
}

func (aa AddAddress) String() string {
	return fmt.Sprintf("ip addr add %s dev %s", aa.Address.String(), aa.InterfaceName)
}
