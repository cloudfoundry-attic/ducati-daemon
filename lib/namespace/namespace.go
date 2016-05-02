package namespace

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

//go:generate counterfeiter -o ../../fakes/namespace.go --fake-name Namespace . Namespace
type Namespace interface {
	Execute(func(*os.File) error) error
	Name() string
	Fd() uintptr
}

type Netns struct {
	*os.File
}

func (n *Netns) String() string {
	var stat unix.Stat_t
	if err := unix.Fstat(int(n.Fd()), &stat); err != nil {
		return fmt.Sprintf("%s:[unknown]", n.Name())
	}
	return fmt.Sprintf("%s:[%d]", n.Name(), stat.Ino)
}

//go:generate counterfeiter -o ../../fakes/opener.go --fake-name Opener . Opener
type Opener interface {
	OpenPath(path string) (Namespace, error)
}

type PathOpener struct{}

func (*PathOpener) OpenPath(path string) (Namespace, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &Netns{
		File: file,
	}, nil
}
