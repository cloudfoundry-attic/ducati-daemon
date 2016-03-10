package commands

import "fmt"

//go:generate counterfeiter --fake-name LinkDeletor . LinkDeletor
type LinkDeletor interface {
	DeleteLinkByName(name string) error
}

type DeleteLink struct {
	LinkName string
}

func (c DeleteLink) Execute(context Context) error {
	err := context.LinkDeletor().DeleteLinkByName(c.LinkName)
	if err != nil {
		return fmt.Errorf("delete link: %s", err)
	}

	return nil
}

func (c DeleteLink) String() string {
	return fmt.Sprintf("ip link del %s", c.LinkName)
}
