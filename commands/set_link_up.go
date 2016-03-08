package commands

import "fmt"

//go:generate counterfeiter --fake-name SetUpper . SetUpper
type SetUpper interface {
	SetUp(name string) error
}

type SetLinkUp struct {
	LinkName string
}

func (s SetLinkUp) Execute(context Context) error {
	err := context.SetUpper().SetUp(s.LinkName)
	if err != nil {
		return fmt.Errorf("set link up: %s", err)
	}

	return nil
}
