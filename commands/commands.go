package commands

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
}

//go:generate counterfeiter --fake-name Condition . Condition
type Condition interface {
	Satisfied(interface{}) bool
}

func All(commands ...Command) Command {
	return Group(commands)
}

type Group []Command

func (g Group) Execute(context Context) error {
	for _, c := range g {
		if err := c.Execute(context); err != nil {
			return err
		}
	}
	return nil
}
