// +build !linux

package namespace

import "os"

func (n *namespace) Execute(callback func(*os.File) error) error {
	f, err := os.Open(n.path)
	if err != nil {
		return err
	}
	defer f.Close()

	return callback(f)
}
