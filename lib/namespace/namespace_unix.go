// +build !linux

package namespace

import "os"

func (n *Netns) Execute(callback func(*os.File) error) error {
	return callback(nil)
}
