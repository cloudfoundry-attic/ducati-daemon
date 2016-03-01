package commands

//go:generate counterfeiter --fake-name SetNamespacer . SetNamespacer
type SetNamespacer interface {
	SetNamespace(intefaceName, namespace string) error
}

type MoveLink struct {
	Name      string
	Namespace string
}

func (s MoveLink) Execute(context Context) error {
	return context.SetNamespacer().SetNamespace(s.Name, s.Namespace)
}
