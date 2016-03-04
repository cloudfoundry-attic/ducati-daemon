package nl

import (
	"syscall"

	"github.com/vishvananda/netlink"
)

const FAMILY_V4 = netlink.FAMILY_V4

//go:generate counterfeiter --fake-name NLSocket . NLSocket
type NLSocket interface {
	Receive() ([]syscall.NetlinkMessage, error)
	Close()
}

//go:generate counterfeiter --fake-name Netlinker . Netlinker
type Netlinker interface {
	LinkAdd(link netlink.Link) error
	LinkDel(link netlink.Link) error
	LinkList() ([]netlink.Link, error)
	LinkSetUp(link netlink.Link) error
	LinkByName(name string) (netlink.Link, error)
	LinkSetNsFd(link netlink.Link, fd int) error
	AddrAdd(link netlink.Link, addr *netlink.Addr) error
	LinkSetMaster(slave netlink.Link, master *netlink.Bridge) error
	LinkByIndex(int) (netlink.Link, error)
	RouteAdd(*netlink.Route) error
	RouteList(netlink.Link, int) ([]netlink.Route, error)
	Subscribe(int, ...uint) (NLSocket, error)
	NeighDeserialize([]byte) (*netlink.Neigh, error)
}
