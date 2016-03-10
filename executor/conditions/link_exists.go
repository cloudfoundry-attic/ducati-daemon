package conditions

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type LinkExists struct {
	Name string
}

func (l LinkExists) Satisfied(context executor.Context) bool {
	return context.LinkFactory().Exists(l.Name)
}

func (l LinkExists) String() string {
	return fmt.Sprintf(`check if link "%s" exists`, l.Name)
}
