package commands

import "fmt"

//go:generate counterfeiter --fake-name Repository . Repository
type Repository interface {
	Get(path string) (Namespace, error)
	Create(name string) (Namespace, error)
}

type CreateNamespace struct {
	Name       string
	Repository Repository
}

func (cn CreateNamespace) Execute(context Context) error {
	if _, err := cn.Repository.Create(cn.Name); err != nil {
		return fmt.Errorf("failed to create namespace %q: %s", cn.Name, err)
	}
	return nil
}
