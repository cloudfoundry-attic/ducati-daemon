package commands

//go:generate counterfeiter --fake-name MasterSetter . MasterSetter
type MasterSetter interface {
	SetMaster(slave, master string) error
}

type SetLinkMaster struct {
	Master string
	Slave  string
}

func (slm SetLinkMaster) Execute(context Context) error {
	return context.MasterSetter().SetMaster(slm.Slave, slm.Master)
}
