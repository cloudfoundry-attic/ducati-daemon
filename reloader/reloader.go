package reloader

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
)

type Reloader struct {
	Watcher watcher.MissWatcher
}

func (r *Reloader) Callback(ns namespace.Namespace) error {

	nsName := ns.Name()

	vxlanName, err := getVxlanName(nsName)
	if err != nil {
		return fmt.Errorf("get vxlan name: %s", err)
	}

	err = r.Watcher.StartMonitor(ns, vxlanName)
	if err != nil {
		return fmt.Errorf("start monitor: %s", err)
	}

	return nil
}

func getVxlanName(nsName string) (string, error) {
	s := strings.TrimPrefix(path.Base(nsName), "vni-")

	if s == nsName {
		return "", errors.New("not a valid sandbox name")
	}

	return "vxlan" + s, nil
}
