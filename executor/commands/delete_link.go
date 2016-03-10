package commands

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type DeleteLink struct {
	LinkName string
}

func (c DeleteLink) Execute(context executor.Context) error {
	err := context.LinkFactory().DeleteLinkByName(c.LinkName)
	if err != nil {
		return fmt.Errorf("delete link: %s", err)
	}

	return nil
}

func (c DeleteLink) String() string {
	return fmt.Sprintf("ip link del %s", c.LinkName)
}
