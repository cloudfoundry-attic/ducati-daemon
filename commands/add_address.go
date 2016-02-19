package commands

import "net"

//go:generate counterfeiter --fake-name AddressAdder . AddressAdder
type AddressAdder interface {
	AddAddress(interfaceName string, address *net.IPNet) error
}

type AddAddress struct {
	InterfaceName string
	Address       net.IPNet
}

func (aa AddAddress) Execute(context Context) error {
	return context.AddressAdder().AddAddress(aa.InterfaceName, &aa.Address)
}
