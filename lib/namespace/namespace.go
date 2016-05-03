package namespace

import (
	"fmt"
	"os"

	"github.com/pivotal-golang/lager"

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
	Logger lager.Logger
}

func (n *Netns) String() string {
	return fmt.Sprintf("%s:[%s]", n.Name(), n.inode())
}

func (n *Netns) inode() string {
	var stat unix.Stat_t

	err := unix.Fstat(int(n.Fd()), &stat)
	if err != nil {
		return "unknown"
	}

	return fmt.Sprintf("%d", stat.Ino)
}

func (n *Netns) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{ "name": "%s", "inode": "%s" }`, n.Name(), n.inode())), nil
}

//go:generate counterfeiter -o ../../fakes/opener.go --fake-name Opener . Opener
type Opener interface {
	OpenPath(path string) (Namespace, error)
}

type PathOpener struct {
	Logger lager.Logger
}

func (po *PathOpener) OpenPath(path string) (Namespace, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &Netns{
		File:   file,
		Logger: po.Logger,
	}, nil
}
