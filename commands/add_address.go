package commands

import (
	"fmt"
	"net"
)

//go:generate counterfeiter --fake-name AddressAdder . AddressAdder
type AddressAdder interface {
	AddAddress(interfaceName string, address *net.IPNet) error
}

type AddAddress struct {
	InterfaceName string
	Address       net.IPNet
}

func (aa AddAddress) Execute(context Context) error {
	err := context.AddressAdder().AddAddress(aa.InterfaceName, &aa.Address)
	if err != nil {
		return fmt.Errorf("add address: %s", err)
	}

	return nil
}

func (aa AddAddress) String() string {
	return fmt.Sprintf("ip addr add %s dev %s", aa.Address.String(), aa.InterfaceName)
}
