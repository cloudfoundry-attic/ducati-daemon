package commands

//go:generate counterfeiter --fake-name Context . Context
type Context interface {
	AddressAdder() AddressAdder
	BridgeFactory() BridgeFactory
	MasterSetter() MasterSetter
	RouteAdder() RouteAdder
	SetUpper() SetUpper
	VethFactory() VethFactory
	VxlanFactory() VxlanFactory
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
	return all{commands: commands}
}

type all struct {
	commands []Command
}

func (a all) Execute(context Context) error {
	for _, c := range a.commands {
		if err := c.Execute(context); err != nil {
			return err
		}
	}
	return nil
}
