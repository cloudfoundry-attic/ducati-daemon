package executor

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/conditions"
)

//go:generate counterfeiter --fake-name AddressManager . AddressManager
type AddressManager interface {
	commands.AddressAdder
}

//go:generate counterfeiter --fake-name RouteManager . RouteManager
type RouteManager interface {
	commands.RouteAdder
}

//go:generate counterfeiter --fake-name LinkFactory . LinkFactory
type LinkFactory interface {
	commands.BridgeFactory
	commands.HardwareAddresser
	commands.MasterSetter
	commands.SetNamespacer
	commands.SetUpper
	commands.VethFactory
	commands.VxlanFactory
	commands.LinkDeletor
	commands.VethDeviceCounter
}

//go:generate counterfeiter --fake-name Executor . Executor
type Executor interface {
	Execute(commands.Command) error
}

//go:generate counterfeiter --fake-name Context . Context
type Context interface {
	conditions.Context
	commands.Context
}

func New(
	addressManager AddressManager,
	routeManager RouteManager,
	linkFactory LinkFactory,
) Executor {
	return &executor{
		AddressManager: addressManager,
		RouteManager:   routeManager,
		LinkFactory:    linkFactory,
	}
}

type executor struct {
	AddressManager AddressManager
	RouteManager   RouteManager
	LinkFactory    LinkFactory
}

func (e *executor) Execute(command commands.Command) error {
	return command.Execute(e)
}

func (e *executor) AddressAdder() commands.AddressAdder {
	return e.AddressManager
}

func (e *executor) RouteAdder() commands.RouteAdder {
	return e.RouteManager
}

func (e *executor) BridgeFactory() commands.BridgeFactory {
	return e.LinkFactory
}

func (e *executor) HardwareAddresser() commands.HardwareAddresser {
	return e.LinkFactory
}

func (e *executor) MasterSetter() commands.MasterSetter {
	return e.LinkFactory
}

func (e *executor) SetNamespacer() commands.SetNamespacer {
	return e.LinkFactory
}

func (e *executor) SetUpper() commands.SetUpper {
	return e.LinkFactory
}

func (e *executor) VethFactory() commands.VethFactory {
	return e.LinkFactory
}

func (e *executor) VxlanFactory() commands.VxlanFactory {
	return e.LinkFactory
}

func (e *executor) VethDeviceCounter() commands.VethDeviceCounter {
	return e.LinkFactory
}

func (e *executor) LinkDeletor() commands.LinkDeletor {
	return e.LinkFactory
}
