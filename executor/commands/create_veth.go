package commands

import "fmt"

type CreateVeth struct {
	Name     string
	PeerName string
	MTU      int
}

func (cv CreateVeth) Execute(context Context) error {
	err := context.LinkFactory().CreateVeth(cv.Name, cv.PeerName, cv.MTU)
	if err != nil {
		return fmt.Errorf("create veth: %s", err)
	}
	return nil
}

func (cs CreateVeth) String() string {
	return fmt.Sprintf("ip link add dev %s mtu %d type veth peer name %s mtu %d", cs.Name, cs.MTU, cs.PeerName, cs.MTU)
}
