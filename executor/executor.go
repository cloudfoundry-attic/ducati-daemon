package executor

import "net"

//go:generate counterfeiter --fake-name AddressManager . AddressManager
type AddressManager interface {
	AddAddress(interfaceName string, address *net.IPNet) error
}

//go:generate counterfeiter --fake-name RouteManager . RouteManager
type RouteManager interface {
	AddRoute(interfaceName string, destination *net.IPNet, gateway net.IP) error
}

//go:generate counterfeiter --fake-name LinkFactory . LinkFactory
type LinkFactory interface {
	CreateBridge(name string) error
	CreateVeth(name, peerName string, mtu int) error
	CreateVxlan(name string, vni int) error
	DeleteLinkByName(name string) error
	Exists(name string) bool
	HardwareAddress(linkName string) (net.HardwareAddr, error)
	SetMaster(slave, master string) error
	SetNamespace(intefaceName, namespace string) error
	SetUp(name string) error
	VethDeviceCount() (int, error)
}

//go:generate counterfeiter --fake-name Command . Command
type Command interface {
	Execute(context Context) error
	String() string
}

//go:generate counterfeiter --fake-name Condition . Condition
type Condition interface {
	Satisfied(context Context) bool
	String() string
}

//go:generate counterfeiter --fake-name Executor . Executor
type Executor interface {
	Execute(Command) error
}

//go:generate counterfeiter --fake-name Context . Context
type Context interface {
	AddressManager() AddressManager
	LinkFactory() LinkFactory
	RouteManager() RouteManager
}

func New(
	addressManager AddressManager,
	routeManager RouteManager,
	linkFactory LinkFactory,
) Executor {
	return &executor{
		addressManager: addressManager,
		routeManager:   routeManager,
		linkFactory:    linkFactory,
	}
}

type executor struct {
	addressManager AddressManager
	routeManager   RouteManager
	linkFactory    LinkFactory
}

func (e *executor) Execute(command Command) error {
	return command.Execute(e)
}

func (e *executor) AddressManager() AddressManager {
	return e.addressManager
}

func (e *executor) RouteManager() RouteManager {
	return e.routeManager
}

func (e *executor) LinkFactory() LinkFactory {
	return e.linkFactory
}
