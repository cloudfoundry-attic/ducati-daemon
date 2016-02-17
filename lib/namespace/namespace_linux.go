package namespace

import (
	"os"

	"github.com/appc/cni/pkg/ns"
)

func (n *namespace) Execute(callback func(*os.File) error) error {
	return ns.WithNetNSPath(n.path, false, func(f *os.File) error {
		return callback(f)
	})
}
