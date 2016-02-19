package commands

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
	return context.VethFactory().CreateVeth(cv.Name, cv.PeerName, cv.MTU)
}
