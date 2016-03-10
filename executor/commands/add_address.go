package commands

import (
	"fmt"
	"net"
)

type AddAddress struct {
	InterfaceName string
	Address       net.IPNet
}

func (aa AddAddress) Execute(context Context) error {
	err := context.AddressManager().AddAddress(aa.InterfaceName, &aa.Address)
	if err != nil {
		return fmt.Errorf("add address: %s", err)
	}

	return nil
}

func (aa AddAddress) String() string {
	return fmt.Sprintf("ip addr add %s dev %s", aa.Address.String(), aa.InterfaceName)
}
