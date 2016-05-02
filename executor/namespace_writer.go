package executor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
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
			output, err := exec.Command("stat", "-L", "-c", "%i", "/proc/self/ns/net").CombinedOutput()
			if err != nil {
				logger.Error("stat", err)
			} else {
				logger.Info("inode", lager.Data{"namespace": strings.TrimSpace(string(output))})
			}

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
