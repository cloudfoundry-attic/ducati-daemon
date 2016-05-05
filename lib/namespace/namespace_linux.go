package namespace

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/appc/cni/pkg/ns"
	"github.com/pivotal-golang/lager"
)

func (n *Netns) Execute(callback func(*os.File) error) error {
	resultCh := make(chan error)

	go func() { resultCh <- n.execute(callback) }()

	err := <-resultCh
	if err != nil {
		n.Logger.Error("execute", err)
	}
	return err
}

func (n *Netns) execute(callback func(*os.File) error) error {
	logger := n.Logger.Session("execute", lager.Data{"namespace": n})

	n.ThreadLocker.LockOSThread()
	//defer n.ThreadLocker.UnlockOSThread()

	originalNamespace, err := os.Open(taskNamespacePath())
	if err != nil {
		return fmt.Errorf("open failed: %s", err)
	}
	defer originalNamespace.Close()

	if err := ns.SetNS(n.File, syscall.CLONE_NEWNET); err != nil {
		return fmt.Errorf("set ns failed: %s", err)
	}
	defer func() {
		if err := ns.SetNS(originalNamespace, syscall.CLONE_NEWNET); err != nil {
			logger.Error("returning to original namespace", err)
			panic(err)
		}
	}()

	logger.Info("invoking-callback")

	// TODO: remove
	buf := make([]byte, 1<<16)
	runtime.Stack(buf, false)
	buf = bytes.Trim(buf, "\x00")
	trace := fmt.Sprintf("%s", buf)
	logger.Info("stacktrace:\n" + trace)

	if err := callback(originalNamespace); err != nil {
		logger.Error("callback-failed", err)
		return err
	}

	logger.Info("callback-complete")
	return nil
}

func taskNamespacePath() string {
	return fmt.Sprintf("/proc/%d/task/%d/ns/net", os.Getpid(), syscall.Gettid())
}
