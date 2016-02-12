package nl

import "github.com/vishvananda/netlink"

type nl struct{}

var Netlink = &nl{}

func (*nl) LinkAdd(link netlink.Link) error {
	return netlink.LinkAdd(link)
}

func (*nl) LinkSetUp(link netlink.Link) error {
	return netlink.LinkSetUp(link)
}

func (*nl) LinkByName(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}

func (*nl) LinkByIndex(index int) (netlink.Link, error) {
	return netlink.LinkByIndex(index)
}

func (*nl) LinkSetNsFd(link netlink.Link, fd int) error {
	return netlink.LinkSetNsFd(link, fd)
}

func (*nl) AddrAdd(link netlink.Link, addr *netlink.Addr) error {
	return netlink.AddrAdd(link, addr)
}

func (*nl) LinkSetMaster(slave netlink.Link, master *netlink.Bridge) error {
	return netlink.LinkSetMaster(slave, master)
}

func (*nl) RouteAdd(route *netlink.Route) error {
	return netlink.RouteAdd(route)
}

func (*nl) LinkDel(link netlink.Link) error {
	return netlink.LinkDel(link)
}

func (*nl) LinkList() ([]netlink.Link, error) {
	return netlink.LinkList()
}

func (*nl) RouteList(link netlink.Link, family int) ([]netlink.Route, error) {
	return netlink.RouteList(link, family)
}
