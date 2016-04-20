package links

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
	vnl "github.com/vishvananda/netlink/nl"
)

const (
	BridgeMTU    = 1500
	VxlanPort    = 4789
	VxlanVethMTU = 1450
)

type netlinker interface {
	LinkAdd(link netlink.Link) error
	LinkByName(name string) (netlink.Link, error)
	LinkDel(link netlink.Link) error
	LinkList() ([]netlink.Link, error)
	LinkSetMaster(slave netlink.Link, master *netlink.Bridge) error
	LinkSetNsFd(link netlink.Link, fd int) error
	LinkSetUp(link netlink.Link) error
}

type Factory struct {
	Netlinker netlinker
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

func (f *Factory) CreateDummy(name string) error {
	dummy := &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
		},
	}

	return f.Netlinker.LinkAdd(dummy)
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

func (f *Factory) CreateVxlan(name string, vni int) error {
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

	return f.Netlinker.LinkAdd(vxlan)
}

func (f *Factory) FindLink(name string) (netlink.Link, error) {
	return f.Netlinker.LinkByName(name)
}

func (f *Factory) Exists(name string) bool {
	if _, err := f.FindLink(name); err != nil {
		return false
	}

	return true
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

func (f *Factory) HardwareAddress(linkName string) (net.HardwareAddr, error) {
	link, err := f.FindLink(linkName)
	if err != nil {
		return nil, fmt.Errorf("failed to find link: %s", err)
	}

	return link.Attrs().HardwareAddr, nil
}

func (f *Factory) ListLinks() ([]netlink.Link, error) {
	return f.Netlinker.LinkList()
}

func (f *Factory) SetMaster(slave, master string) error {
	link, err := f.FindLink(master)
	if err != nil {
		return fmt.Errorf("failed to find master: %s", err)
	}

	bridge, ok := link.(*netlink.Bridge)
	if !ok {
		return fmt.Errorf("master must be a bridge")
	}

	link, err = f.FindLink(slave)
	if err != nil {
		return fmt.Errorf("failed to find slave: %s", err)
	}

	err = f.Netlinker.LinkSetMaster(link, bridge)
	if err != nil {
		return fmt.Errorf("failed to set master: %s", err)
	}

	return nil
}

func (f *Factory) SetNamespace(linkName string, fd uintptr) error {
	link, err := f.FindLink(linkName)
	if err != nil {
		return fmt.Errorf("failed to find link: %s", err)
	}

	err = f.Netlinker.LinkSetNsFd(link, int(fd))
	if err != nil {
		return fmt.Errorf("failed to set link namespace: %s", err)
	}

	return nil
}

func (f *Factory) SetUp(name string) error {
	link, err := f.FindLink(name)
	if err != nil {
		return err
	}

	if err := f.Netlinker.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to set link up: %s", err)
	}

	return nil
}

func (f *Factory) VethDeviceCount() (int, error) {
	count := 0

	links, err := f.Netlinker.LinkList()
	if err != nil {
		return 0, fmt.Errorf("failed to list links: %s", err)
	}
	for _, link := range links {
		if _, ok := link.(*netlink.Veth); ok {
			count++
		}
	}

	return count, nil
}
