package commands

import (
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

type CreateVxlan struct {
	Name string
	VNI  int
}

func (cv CreateVxlan) Execute(context executor.Context) error {
	err := context.LinkFactory().CreateVxlan(cv.Name, cv.VNI)
	if err != nil {
		return fmt.Errorf("create vxlan: %s", err)
	}

	return nil
}

func (cv CreateVxlan) String() string {
	return fmt.Sprintf("ip link add %s type vxlan vni %d", cv.Name, cv.VNI)
}
