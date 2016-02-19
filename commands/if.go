package commands

type If struct {
	Condition Condition
	Command   Command
}

func (i If) Execute(context Context) error {
	if !i.Condition.Satisfied(context) {
		return nil
	}

	return i.Command.Execute(context)
}
