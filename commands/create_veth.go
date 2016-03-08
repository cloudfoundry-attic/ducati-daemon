package commands

import "fmt"

//go:generate counterfeiter --fake-name VethFactory . VethFactory
type VethFactory interface {
	CreateVeth(name, peerName string, mtu int) error
}

type CreateVeth struct {
	Name     string
	PeerName string
	MTU      int
}

func (cv CreateVeth) Execute(context Context) error {
	err := context.VethFactory().CreateVeth(cv.Name, cv.PeerName, cv.MTU)
	if err != nil {
		return fmt.Errorf("create veth: %s", err)
	}
	return nil
}
