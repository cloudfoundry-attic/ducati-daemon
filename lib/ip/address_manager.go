package ip

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

type netlinker interface {
	AddrAdd(link netlink.Link, addr *netlink.Addr) error
	LinkByName(name string) (netlink.Link, error)
	RouteAdd(*netlink.Route) error
}

type AddressManager struct {
	Netlinker netlinker
}

func (am *AddressManager) AddAddress(interfaceName string, address *net.IPNet) error {
	link, err := am.Netlinker.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("link by name failed: %s", err)
	}

	err = am.Netlinker.AddrAdd(link, &netlink.Addr{IPNet: address})
	if err != nil {
		return fmt.Errorf("address add failed: %s", err)
	}

	return nil
}
