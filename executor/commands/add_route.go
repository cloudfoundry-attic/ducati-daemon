package commands

import (
	"fmt"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type AddRoute struct {
	Interface   string
	Destination net.IPNet
	Gateway     net.IP
}

func (ad AddRoute) Execute(context executor.Context) error {
	err := context.RouteManager().AddRoute(ad.Interface, &ad.Destination, ad.Gateway)
	if err != nil {
		return fmt.Errorf("add route: %s", err)
	}

	return nil
}

func (ad AddRoute) String() string {
	return fmt.Sprintf("ip route add dev %s %s via %s", ad.Interface, ad.Destination.String(), ad.Gateway)
}
