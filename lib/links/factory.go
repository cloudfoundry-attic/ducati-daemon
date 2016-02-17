package links

import (
	"fmt"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/nl"
	"github.com/vishvananda/netlink"
	vnl "github.com/vishvananda/netlink/nl"
)

const (
	BridgeMTU    = 1500
	VxlanPort    = 4789
	VxlanVethMTU = 1450
)

type Factory struct {
	Netlinker nl.Netlinker
}

func (f *Factory) CreateBridge(name string, addr *net.IPNet) (*netlink.Bridge, error) {
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
			MTU:  BridgeMTU,
		},
	}

	err := f.Netlinker.LinkAdd(bridge)
	if err != nil {
		return nil, err
	}

	err = f.Netlinker.AddrAdd(bridge, &netlink.Addr{IPNet: addr})
	if err != nil {
		return nil, err
	}

	err = f.Netlinker.LinkSetUp(bridge)
	if err != nil {
		return nil, err
	}

	return bridge, nil
}

func (f *Factory) CreateVethPair(containerID, hostIfaceName string, mtu int) error {
	if len(containerID) > 11 {
		containerID = containerID[:11]
	}

	containerLink := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: hostIfaceName,
			MTU:  mtu,
		},
		PeerName: containerID,
	}

	err := f.Netlinker.LinkAdd(containerLink)
	if err != nil {
		return fmt.Errorf("link add: %s", err)
	}

	return nil
}

func (f *Factory) CreateVxlan(name string, vni int) (netlink.Link, error) {
	vxlan := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
		},
		VxlanId:  vni,
		Learning: true,
		Port:     int(vnl.Swap16(VxlanPort)), //network endian order
		Proxy:    true,
		L3miss:   true,
		L2miss:   true,
	}

	err := f.Netlinker.LinkAdd(vxlan)
	if err != nil {
		return nil, err
	}

	return vxlan, nil
}

func (f *Factory) FindLink(name string) (netlink.Link, error) {
	return f.Netlinker.LinkByName(name)
}

func (f *Factory) DeleteLink(link netlink.Link) error {
	return f.Netlinker.LinkDel(link)
}

func (f *Factory) DeleteLinkByName(name string) error {
	link, err := f.Netlinker.LinkByName(name)
	if err != nil {
		return err
	}

	err = f.Netlinker.LinkDel(link)
	if err != nil {
		return err
	}

	return nil
}

func (f *Factory) ListLinks() ([]netlink.Link, error) {
	return f.Netlinker.LinkList()
}
