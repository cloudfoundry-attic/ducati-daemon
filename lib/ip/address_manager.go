package ip

import (
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/nl"
	"github.com/vishvananda/netlink"
)

type AddressManager struct {
	Netlinker nl.Netlinker
}

func (am *AddressManager) AddAddress(link netlink.Link, address *net.IPNet) error {
	addr := &netlink.Addr{
		IPNet: address,
	}
	return am.Netlinker.AddrAdd(link, addr)
}
