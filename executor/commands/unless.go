package commands

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type Unless struct {
	Condition executor.Condition
	Command   executor.Command
}

func (u Unless) Execute(context executor.Context) error {
	if u.Condition.Satisfied(context) {
		return nil
	}

	err := u.Command.Execute(context)
	if err != nil {
		return fmt.Errorf("unless: %s", err)
	}

	return nil
}

func (u Unless) String() string {
	return fmt.Sprintf("%s || %s", u.Condition, u.Command)
}
