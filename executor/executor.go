package executor

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/conditions"
)

//go:generate counterfeiter --fake-name AddressManager . AddressManager
type AddressManager interface {
	commands.AddressManager
}

//go:generate counterfeiter --fake-name RouteManager . RouteManager
type RouteManager interface {
	commands.RouteManager
}

//go:generate counterfeiter --fake-name LinkFactory . LinkFactory
type LinkFactory interface {
	commands.LinkFactory
}

//go:generate counterfeiter --fake-name Executor . Executor
type Executor interface {
	Execute(commands.Command) error
}

//go:generate counterfeiter -o ../fakes/context.go --fake-name Context . Context
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

func (e *executor) Execute(command commands.Command) error {
	return command.Execute(e)
}

func (e *executor) AddressManager() commands.AddressManager {
	return e.addressManager
}

func (e *executor) RouteManager() commands.RouteManager {
	return e.routeManager
}

func (e *executor) LinkFactory() commands.LinkFactory {
	return e.linkFactory
}
