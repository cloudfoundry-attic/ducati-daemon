package commands

import "fmt"

//go:generate counterfeiter --fake-name MasterSetter . MasterSetter
type MasterSetter interface {
	SetMaster(slave, master string) error
}

type SetLinkMaster struct {
	Master string
	Slave  string
}

func (slm SetLinkMaster) Execute(context Context) error {
	err := context.MasterSetter().SetMaster(slm.Slave, slm.Master)
	if err != nil {
		return fmt.Errorf("set link master: %s", err)
	}

	return nil
}
