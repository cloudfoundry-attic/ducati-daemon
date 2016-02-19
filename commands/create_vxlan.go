package commands

//go:generate counterfeiter --fake-name VxlanFactory  . VxlanFactory
type VxlanFactory interface {
	CreateVxlan(name string, vni int) error
}

type CreateVxlan struct {
	Name string
	VNI  int
}

func (cv CreateVxlan) Execute(context Context) error {
	return context.VxlanFactory().CreateVxlan(cv.Name, cv.VNI)
}
