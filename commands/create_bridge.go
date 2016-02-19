package commands

//go:generate counterfeiter --fake-name BridgeFactory  . BridgeFactory
type BridgeFactory interface {
	CreateBridge(name string) error
}

type CreateBridge struct {
	Name string
}

func (cb CreateBridge) Execute(context Context) error {
	return context.BridgeFactory().CreateBridge(cb.Name)
}
