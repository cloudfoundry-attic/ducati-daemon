package executor

import (
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/sandbox"
	"github.com/tedsuo/ifrit"
)

//go:generate counterfeiter -o ../fakes/address_manager.go --fake-name AddressManager . AddressManager
type AddressManager interface {
	AddAddress(interfaceName string, address *net.IPNet) error
}

//go:generate counterfeiter -o ../fakes/route_manager.go --fake-name RouteManager . RouteManager
type RouteManager interface {
	AddRoute(interfaceName string, destination *net.IPNet, gateway net.IP) error
}

//go:generate counterfeiter -o ../fakes/link_factory.go --fake-name LinkFactory . LinkFactory
type LinkFactory interface {
	CreateBridge(name string) error
	CreateDummy(name string) error
	CreateVeth(name, peerName string, mtu int) error
	CreateVxlan(name string, vni int) error
	DeleteLinkByName(name string) error
	Exists(name string) bool
	HardwareAddress(linkName string) (net.HardwareAddr, error)
	SetMaster(slave, master string) error
	SetNamespace(intefaceName string, fd uintptr) error
	SetUp(name string) error
	VethDeviceCount() (int, error)
}

//go:generate counterfeiter -o ../fakes/listener_factory.go --fake-name ListenerFactory . ListenerFactory
type ListenerFactory interface {
	ListenUDP(network string, address *net.UDPAddr) (*net.UDPConn, error)
}

type ListenUDPFunc func(network string, address *net.UDPAddr) (*net.UDPConn, error)

func (l ListenUDPFunc) ListenUDP(network string, address *net.UDPAddr) (*net.UDPConn, error) {
	return l(network, address)
}

//go:generate counterfeiter -o ../fakes/dns_server_factory.go --fake-name DNSServerFactory . DNSServerFactory
type DNSServerFactory interface {
	New(listener net.PacketConn) ifrit.Runner
}

//go:generate counterfeiter -o ../fakes/command.go --fake-name Command . Command
type Command interface {
	Execute(context Context) error
	String() string
}

//go:generate counterfeiter -o ../fakes/condition.go --fake-name Condition . Condition
type Condition interface {
	Satisfied(context Context) (bool, error)
	String() string
}

//go:generate counterfeiter -o ../fakes/executor.go --fake-name Executor . Executor
type Executor interface {
	Execute(Command) error
}

//go:generate counterfeiter -o ../fakes/context.go --fake-name Context . Context
type Context interface {
	AddressManager() AddressManager
	LinkFactory() LinkFactory
	RouteManager() RouteManager
	SandboxNamespaceRepository() namespace.Repository
	SandboxRepository() sandbox.Repository
	ListenerFactory() ListenerFactory
	DNSServerFactory() DNSServerFactory
}

func New(
	addressManager AddressManager,
	routeManager RouteManager,
	linkFactory LinkFactory,
	sandboxNamespaceRepository namespace.Repository,
	sandboxRepository sandbox.Repository,
	listenerFactory ListenerFactory,
	dnsServerFactory DNSServerFactory,
) Executor {
	return &executor{
		addressManager:             addressManager,
		routeManager:               routeManager,
		linkFactory:                linkFactory,
		sandboxNamespaceRepository: sandboxNamespaceRepository,
		sandboxRepository:          sandboxRepository,
		listenerFactory:            listenerFactory,
		dnsServerFactory:           dnsServerFactory,
	}
}

type executor struct {
	addressManager             AddressManager
	routeManager               RouteManager
	linkFactory                LinkFactory
	sandboxNamespaceRepository namespace.Repository
	sandboxRepository          sandbox.Repository
	listenerFactory            ListenerFactory
	dnsServerFactory           DNSServerFactory
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

func (e *executor) SandboxNamespaceRepository() namespace.Repository {
	return e.sandboxNamespaceRepository
}

func (e *executor) SandboxRepository() sandbox.Repository {
	return e.sandboxRepository
}

func (e *executor) ListenerFactory() ListenerFactory {
	return e.listenerFactory
}

func (e *executor) DNSServerFactory() DNSServerFactory {
	return e.dnsServerFactory
}
