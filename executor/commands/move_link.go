package commands

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type MoveLink struct {
	Name      string
	Namespace string
}

func (s MoveLink) Execute(context executor.Context) error {
	err := context.LinkFactory().SetNamespace(s.Name, s.Namespace)
	if err != nil {
		return fmt.Errorf("move link: %s", err)
	}

	return nil
}

func (s MoveLink) String() string {
	return fmt.Sprintf("ip link set dev %s netns %s", s.Name, s.Namespace)
}