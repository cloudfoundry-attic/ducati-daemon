// +build linux

package namespace

import (
	"os"

	"golang.org/x/sys/unix"
)

func unlinkNetworkNamespace(path string) error {
	if err := unix.Unmount(path, unix.MNT_DETACH); err != nil {
		return err
	}
	return os.Remove(path)
}

func bindMountFile(src, dst string) error {
	// mount point has to be an existing file
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	f.Close()

	return unix.Mount(src, dst, "none", unix.MS_BIND, "")
}
