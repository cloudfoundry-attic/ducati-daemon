package neigh

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/nl"
	"github.com/cloudfoundry-incubator/ducati-daemon/ossupport"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
	"github.com/pivotal-golang/lager"
	"github.com/vishvananda/netlink"
)

type ARPInserter struct {
	Logger         lager.Logger
	Netlinker      nl.Netlinker
	OSThreadLocker ossupport.OSThreadLocker
}

func (a *ARPInserter) HandleResolvedNeighbors(ns namespace.Executor, resolvedChan <-chan watcher.Neighbor) error {
	ready := make(chan error)

	go a.runLocked(ready, ns, resolvedChan)

	err := <-ready
	if err != nil {
		return fmt.Errorf("namespace execute failed: %s", err)
	}

	return nil
}

func (a *ARPInserter) runLocked(ready chan error, ns namespace.Executor, resolvedChan <-chan watcher.Neighbor) {
	a.OSThreadLocker.LockOSThread()
	defer a.OSThreadLocker.UnlockOSThread()

	err := ns.Execute(func(f *os.File) error {
		close(ready)
		a.addNeighbors(resolvedChan)
		return nil
	})

	if err != nil {
		ready <- err
	}
}

func (a *ARPInserter) addNeighbors(resolvedChan <-chan watcher.Neighbor) {
	for msg := range resolvedChan {
		err := a.Netlinker.AddNeigh(reverseConvert(msg.Neigh))
		if err != nil {
			a.Logger.Error("add-neighbor-failed", err)
		}
	}
}

func reverseConvert(input watcher.Neigh) *netlink.Neigh {
	return &netlink.Neigh{
		LinkIndex:    input.LinkIndex,
		Family:       input.Family,
		State:        input.State,
		Type:         input.Type,
		Flags:        input.Flags,
		IP:           input.IP,
		HardwareAddr: input.HardwareAddr,
	}
}
