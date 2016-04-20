package commands

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type CreateDummy struct {
	Name string
}

func (cv CreateDummy) Execute(context executor.Context) error {
	err := context.LinkFactory().CreateDummy(cv.Name)
	if err != nil {
		return fmt.Errorf("create dummy: %s", err)
	}

	return nil
}

func (cv CreateDummy) String() string {
	return fmt.Sprintf("ip link add %s type dummy", cv.Name)
}
