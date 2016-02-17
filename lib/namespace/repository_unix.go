// +build !linux

package namespace

import "os"

func unlinkNetworkNamespace(path string) error {
	return os.Remove(path)
}

func bindMountFile(src, dst string) error {
	// mount point has to be an existing file
	if f, err := os.Create(dst); err != nil {
		return err
	} else {
		f.Close()
	}

	return nil
}
