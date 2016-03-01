package commands

//go:generate counterfeiter --fake-name LinkDeletor . LinkDeletor
type LinkDeletor interface {
	DeleteLinkByName(name string) error
}

type DeleteLink struct {
	LinkName string
}

func (c DeleteLink) Execute(context Context) error {
	return context.LinkDeletor().DeleteLinkByName(c.LinkName)
}
