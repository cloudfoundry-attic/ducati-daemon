package commands

import "fmt"

type CreateBridge struct {
	Name string
}

func (cb CreateBridge) Execute(context Context) error {
	err := context.LinkFactory().CreateBridge(cb.Name)
	if err != nil {
		return fmt.Errorf("create bridge: %s", err)
	}

	return nil
}

func (cb CreateBridge) String() string {
	return fmt.Sprintf("ip link add dev %s type bridge", cb.Name)
}
