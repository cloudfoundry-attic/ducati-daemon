package namespace

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/ossupport"
	"github.com/pivotal-golang/lager"

	"golang.org/x/sys/unix"
)

type Namespace interface {
	Execute(func(*os.File) error) error
	Name() string
	Fd() uintptr
}

//go:generate counterfeiter -o ../../fakes/namespace.go --fake-name Namespace . jsonNamespace
type jsonNamespace interface {
	Namespace
	MarshalJSON() ([]byte, error)
}

type Netns struct {
	*os.File
	Logger       lager.Logger
	ThreadLocker ossupport.OSThreadLocker
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
	Logger       lager.Logger
	ThreadLocker ossupport.OSThreadLocker
}

func (po *PathOpener) OpenPath(path string) (Namespace, error) {
	logger := po.Logger.Session("open-path")
	logger.Info("opening", lager.Data{"path": path})

	file, err := os.Open(path)
	if err != nil {
		logger.Error("open-failed", err)
		return nil, err
	}

	ns := &Netns{
		Logger:       po.Logger,
		File:         file,
		ThreadLocker: po.ThreadLocker,
	}

	logger.Info("complete", lager.Data{"namespace": ns})

	return ns, nil
}
