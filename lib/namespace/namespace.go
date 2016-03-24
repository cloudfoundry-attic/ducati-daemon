package namespace

import "os"

type Executor interface {
	Execute(func(*os.File) error) error
	Name() string
}

//go:generate counterfeiter -o ../../fakes/namespace.go --fake-name Namespace . Namespace
type Namespace interface {
	Executor
}

type Netns struct {
	*os.File
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
