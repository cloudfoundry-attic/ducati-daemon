package commands

import (
	"bytes"
	"fmt"
	"net"
	"strings"
)

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
	HardwareAddress(linkName string) (net.HardwareAddr, error)
	SetMaster(slave, master string) error
	SetNamespace(intefaceName, namespace string) error
	SetUp(name string) error
	VethDeviceCount() (int, error)
}

//go:generate counterfeiter --fake-name Context . Context
type Context interface {
	AddressManager() AddressManager
	LinkFactory() LinkFactory
	RouteManager() RouteManager
}

//go:generate counterfeiter --fake-name Command . Command
type Command interface {
	Execute(context Context) error
	String() string
}

func All(commands ...Command) Command {
	return Group(commands)
}

type Group []Command

func (g Group) Execute(context Context) error {
	for i, c := range g {
		err := c.Execute(context)
		if err != nil {
			return &GroupError{
				index: i,
				group: g,
				Err:   err,
			}
		}
	}
	return nil
}

type GroupError struct {
	index int
	group Group
	Err   error
}

func (ge *GroupError) Error() string {
	return fmt.Sprintf("%s: commands: %s", ge.Err.Error(), toString(ge.group, ge.index))
}

func (g Group) String() string {
	return toString(g, -1)
}

func toString(group Group, cursor int) string {
	var buffer bytes.Buffer

	buffer.WriteString("(\n")
	for i, command := range group {
		cmdStr := command.String()
		if _, isGroup := command.(Group); isGroup {
			cmdStr = strings.Replace(cmdStr, "\n", "\n    ", -1)
		}

		if i == cursor {
			buffer.WriteString(fmt.Sprintf("--> %s", cmdStr))
		} else {
			buffer.WriteString(fmt.Sprintf("    %s", cmdStr))
		}

		if i < len(group)-1 {
			buffer.WriteString(" &&")
		}

		buffer.WriteString("\n")
	}
	buffer.WriteString(")")

	return buffer.String()
}
