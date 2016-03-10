package commands

import "fmt"

type DeleteLink struct {
	LinkName string
}

func (c DeleteLink) Execute(context Context) error {
	err := context.LinkFactory().DeleteLinkByName(c.LinkName)
	if err != nil {
		return fmt.Errorf("delete link: %s", err)
	}

	return nil
}

func (c DeleteLink) String() string {
	return fmt.Sprintf("ip link del %s", c.LinkName)
}
