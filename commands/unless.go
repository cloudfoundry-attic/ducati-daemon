package commands

type Unless struct {
	Condition Condition
	Command   Command
}

func (u Unless) Execute(context Context) error {
	if u.Condition.Satisfied(context) {
		return nil
	}

	return u.Command.Execute(context)
}
