package ip

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

type RouteManager struct {
	Netlinker netlinker
}

func (rm *RouteManager) AddRoute(interfaceName string, network *net.IPNet, gateway net.IP) error {
	link, err := rm.Netlinker.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("link by name failed: %s", err)
	}

	err = rm.Netlinker.RouteAdd(&netlink.Route{
		LinkIndex: link.Attrs().Index,
		Scope:     netlink.SCOPE_UNIVERSE,
		Dst:       network,
		Gw:        gateway,
	})
	if err != nil {
		return fmt.Errorf("route add failed: %s", err)
	}

	return nil
}
