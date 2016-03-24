// +build !linux

package namespace

import "os"

func bindMountFile(src, dst string) (*os.File, error) {
	return os.Create(dst)
}

func unlinkNetworkNamespace(path string) error {
	return os.Remove(path)
}
