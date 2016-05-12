package commands

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type MoveLink struct {
	Name        string
	SandboxName string
}

func (s MoveLink) Execute(context executor.Context) error {
	sbox, err := context.SandboxRepository().Get(s.SandboxName)
	if err != nil {
		return fmt.Errorf("get sandbox: %s", err)
	}

	err = context.LinkFactory().SetNamespace(s.Name, sbox.Namespace().Fd())
	if err != nil {
		return fmt.Errorf("move link: %s", err)
	}

	return nil
}

func (s MoveLink) String() string {
	return fmt.Sprintf("ip link set dev %s netns %s", s.Name, s.SandboxName)
}
