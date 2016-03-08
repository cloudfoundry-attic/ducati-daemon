package commands

import (
	"fmt"
	"net"
)

//go:generate counterfeiter --fake-name RouteAdder . RouteAdder
type RouteAdder interface {
	AddRoute(interfaceName string, destination *net.IPNet, gateway net.IP) error
}

type AddRoute struct {
	Interface   string
	Destination net.IPNet
	Gateway     net.IP
}

func (ad AddRoute) Execute(context Context) error {
	err := context.RouteAdder().AddRoute(ad.Interface, &ad.Destination, ad.Gateway)
	if err != nil {
		return fmt.Errorf("add route: %s", err)
	}

	return nil
}
