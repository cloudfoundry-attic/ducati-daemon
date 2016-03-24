package namespace

import (
	"os"

	"github.com/appc/cni/pkg/ns"
)

func (n *Netns) Execute(callback func(*os.File) error) error {
	return ns.WithNetNS(n.File, false, func(f *os.File) error {
		return callback(f)
	})
}
