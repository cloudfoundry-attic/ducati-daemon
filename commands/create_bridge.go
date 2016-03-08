package commands

import "fmt"

//go:generate counterfeiter --fake-name BridgeFactory  . BridgeFactory
type BridgeFactory interface {
	CreateBridge(name string) error
}

type CreateBridge struct {
	Name string
}

func (cb CreateBridge) Execute(context Context) error {
	err := context.BridgeFactory().CreateBridge(cb.Name)
	if err != nil {
		return fmt.Errorf("create bridge: %s", err)
	}

	return nil
}
