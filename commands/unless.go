package commands

import "fmt"

type Unless struct {
	Condition Condition
	Command   Command
}

func (u Unless) Execute(context Context) error {
	if u.Condition.Satisfied(context) {
		return nil
	}

	err := u.Command.Execute(context)
	if err != nil {
		return fmt.Errorf("unless: %s", err)
	}

	return nil
}
