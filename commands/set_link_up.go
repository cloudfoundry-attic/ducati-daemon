package commands

//go:generate counterfeiter --fake-name SetUpper . SetUpper
type SetUpper interface {
	SetUp(name string) error
}

type SetLinkUp struct {
	LinkName string
}

func (s SetLinkUp) Execute(context Context) error {
	return context.SetUpper().SetUp(s.LinkName)
}
