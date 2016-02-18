package ip

import (
	"fmt"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/nl"
	"github.com/vishvananda/netlink"
)

type AddressManager struct {
	Netlinker nl.Netlinker
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
