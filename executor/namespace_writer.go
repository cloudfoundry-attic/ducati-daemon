package executor

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
)

type NamespaceWriter struct {
	Namespace namespace.Namespace
	Writer    writer
}

func (nsw *NamespaceWriter) Write(contents []byte) (int, error) {
	var bytesWritten int
	var err error

	nsErr := nsw.Namespace.Execute(func(*os.File) error {
		bytesWritten, err = nsw.Writer.Write(contents)
		return nil
	})
	if nsErr != nil {
		return 0, fmt.Errorf("namespace execute: %s", nsErr)
	}

	return bytesWritten, err
}
