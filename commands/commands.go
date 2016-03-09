package commands

import (
	"bytes"
	"fmt"
	"strings"
)

//go:generate counterfeiter --fake-name Context . Context
type Context interface {
	AddressAdder() AddressAdder
	BridgeFactory() BridgeFactory
	HardwareAddresser() HardwareAddresser
	MasterSetter() MasterSetter
	RouteAdder() RouteAdder
	SetNamespacer() SetNamespacer
	SetUpper() SetUpper
	VethFactory() VethFactory
	VxlanFactory() VxlanFactory
	LinkDeletor() LinkDeletor
	VethDeviceCounter() VethDeviceCounter
}

//go:generate counterfeiter --fake-name Command . Command
type Command interface {
	Execute(context Context) error
	String() string
}

//go:generate counterfeiter --fake-name Condition . Condition
type Condition interface {
	Satisfied(interface{}) bool
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
