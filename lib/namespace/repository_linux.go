package namespace

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func unlinkNetworkNamespace(path string) error {
	if err := unix.Unmount(path, unix.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount: %s", err)
	}
	return os.Remove(path)
}

func bindMountFile(src, dst string) (*os.File, error) {
	// mount point has to be an existing file
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return nil, err
	}
	f.Close()

	err = unix.Mount(src, dst, "none", unix.MS_BIND, "")
	if err != nil {
		return nil, fmt.Errorf("mount: %s")
	}

	return os.Open(dst)
}
