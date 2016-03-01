package commands

//go:generate counterfeiter --fake-name Locker . Locker
type Locker interface {
	Lock(string)
	Unlock(string)
}

type CleanupSandbox struct {
	Namespace  Namespace
	Repository repository
	Locker     Locker
}

func (c CleanupSandbox) Execute(context Context) error {
	panic("not implemented")
}
