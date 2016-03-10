package commands

import "fmt"

type SetLinkUp struct {
	LinkName string
}

func (s SetLinkUp) Execute(context Context) error {
	err := context.LinkFactory().SetUp(s.LinkName)
	if err != nil {
		return fmt.Errorf("set link up: %s", err)
	}

	return nil
}

func (s SetLinkUp) String() string {
	return fmt.Sprintf("ip link set %s up", s.LinkName)
}
