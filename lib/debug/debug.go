package debug

import (
	"fmt"
	"os"
	"syscall"

	"github.com/pivotal-golang/lager"
	"golang.org/x/sys/unix"
)

func taskNamespacePath() string {
	return fmt.Sprintf("/proc/%d/task/%d/ns/net", os.Getpid(), syscall.Gettid())
}

var orig string

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
	orig = inode(hostNamespace)
}

func NetNS() lager.Data {
	var f, err = os.Open(taskNamespacePath())
	if err != nil {
		panic(err)
	}
	defer f.Close()
	curInode := inode(f)

	return lager.Data{"netns-inode": curInode, "isHost": curInode == orig}
}
