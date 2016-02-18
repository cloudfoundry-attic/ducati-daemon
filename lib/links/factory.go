package links

import (
	"fmt"

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

func (f *Factory) CreateBridge(name string) error {
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
			MTU:  BridgeMTU,
		},
	}

	return f.Netlinker.LinkAdd(bridge)
}

func (f *Factory) CreateVeth(name, peerName string, mtu int) error {
	vethLink := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
			MTU:  mtu,
		},
		PeerName: peerName,
	}

	err := f.Netlinker.LinkAdd(vethLink)
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

func (f *Factory) SetMaster(slave, master string) error {
	link, err := f.FindLink(master)
	if err != nil {
		return err
	}

	bridge, ok := link.(*netlink.Bridge)
	if !ok {
		return fmt.Errorf("master must be a bridge")
	}

	link, err = f.FindLink(slave)
	if err != nil {
		return err
	}

	err = f.Netlinker.LinkSetMaster(link, bridge)
	if err != nil {
		return fmt.Errorf("failed to set master: %s", err)
	}

	return nil
}
