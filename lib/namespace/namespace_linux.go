package namespace

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/appc/cni/pkg/ns"
	"github.com/pivotal-golang/lager"
)

var hostNamespaceInode string

func inode(f *os.File) string {
	var stat unix.Stat_t

	err := unix.Fstat(int(f.Fd()), &stat)
	if err != nil {
		return "unknown"
	}

	return fmt.Sprintf("%d", stat.Ino)
}

func init() {
	var hostNamespace, err = os.Open(taskNamespacePath())
	if err != nil {
		panic(err)
	}
	defer hostNamespace.Close()
	hostNamespaceInode = inode(hostNamespace)
}

func (n *Netns) Execute(callback func(*os.File) error) error {
	resultCh := make(chan error)

	go func() {
		n.ThreadLocker.LockOSThread()
		resultCh <- n.execute(callback)
	}()

	err := <-resultCh
	if err != nil {
		n.Logger.Error("execute", err)
	}
	return err
}

func (n *Netns) execute(callback func(*os.File) error) error {
	logger := n.Logger.Session("execute", lager.Data{"namespace": n, "thread": os.Getpid()})

	originalNamespace, err := os.Open(taskNamespacePath())
	if err != nil {
		return fmt.Errorf("open failed: %s", err)
	}
	defer originalNamespace.Close()

	originalNamespaceInode := inode(originalNamespace)
	if originalNamespaceInode != hostNamespaceInode {
		logger.Info("error-original-netns-mismatch", lager.Data{
			"local":  originalNamespaceInode,
			"global": hostNamespaceInode,
		})
	}

	logger.Info("ns-set")
	if err := ns.SetNS(n.File, syscall.CLONE_NEWNET); err != nil {
		return fmt.Errorf("set ns failed: %s", err)
	}
	defer func() {
		logger.Info("ns-restore", lager.Data{"restore-to-inode": originalNamespaceInode})
		if err := ns.SetNS(originalNamespace, syscall.CLONE_NEWNET); err != nil {
			logger.Error("restore", err)
			panic(err)
		}
	}()

	logger.Info("invoking-callback")
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
