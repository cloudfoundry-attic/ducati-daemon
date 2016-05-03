package namespace

import (
	"os"

	"github.com/appc/cni/pkg/ns"
	"github.com/pivotal-golang/lager"
)

func (n *Netns) Execute(callback func(*os.File) error) error {
	logger := n.Logger.Session("execute", lager.Data{"namespace": n})

	return ns.WithNetNS(n.File, false, func(f *os.File) error {
		logger.Info("callback-invoked")

		err := callback(f)
		if err != nil {
			logger.Error("callback-failed", err)
		}

		logger.Info("callback-complete")
		return err
	})
}
