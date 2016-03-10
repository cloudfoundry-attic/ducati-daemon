package commands

import "fmt"

//go:generate counterfeiter --fake-name VxlanFactory  . VxlanFactory
type VxlanFactory interface {
	CreateVxlan(name string, vni int) error
}

type CreateVxlan struct {
	Name string
	VNI  int
}

func (cv CreateVxlan) Execute(context Context) error {
	err := context.VxlanFactory().CreateVxlan(cv.Name, cv.VNI)
	if err != nil {
		return fmt.Errorf("create vxlan: %s", err)
	}

	return nil
}

func (cv CreateVxlan) String() string {
	return fmt.Sprintf("ip link add %s type vxlan vni %d", cv.Name, cv.VNI)
}
