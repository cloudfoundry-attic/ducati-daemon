package executor

import (
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/pivotal-golang/lager"
)

type NamespaceWriter struct {
	Namespace namespace.Namespace
	Writer    writer
}

var logger = lager.NewLogger("namespace-writer")

func (nsw *NamespaceWriter) Write(contents []byte) (int, error) {
	var bytesWritten int
	var err, nsErr error

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		runtime.LockOSThread()
		nsErr = nsw.Namespace.Execute(func(*os.File) error {
			bytesWritten, err = nsw.Writer.Write(contents)
			return nil
		})
		wg.Done()
	}()

	wg.Wait()

	if nsErr != nil {
		return 0, fmt.Errorf("namespace execute: %s", nsErr)
	}

	return bytesWritten, err
}
