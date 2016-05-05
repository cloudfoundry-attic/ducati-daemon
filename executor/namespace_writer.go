package executor

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/pivotal-golang/lager"
)

type NamespaceWriter struct {
	Logger    lager.Logger
	Namespace namespace.Namespace
	Writer    writer
}

func (nsw *NamespaceWriter) Write(contents []byte) (int, error) {
	logger := nsw.Logger.Session("write", lager.Data{"namespace": nsw.Namespace})
	logger.Info("write-called")
	defer logger.Info("write-complete")

	var bytesWritten int
	var err, nsErr error
	nsErr = nsw.Namespace.Execute(func(*os.File) error {
		bytesWritten, err = nsw.Writer.Write(contents)
		return nil
	})

	if nsErr != nil {
		logger.Error("namespace-execute-failed", nsErr)
		return 0, fmt.Errorf("namespace execute: %s", nsErr)
	}

	return bytesWritten, err
}
