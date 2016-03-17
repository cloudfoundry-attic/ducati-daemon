package neigh

import (
	"fmt"
	"os"
	"syscall"

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
		neigh := reverseConvert(msg.Neigh)
		neigh.State = netlink.NUD_REACHABLE

		err := a.Netlinker.SetNeigh(neigh)
		if err != nil {
			a.Logger.Error("set-l3-neighbor-failed", err)
			continue
		}

		fdb := reverseConvert(msg.Neigh)
		fdb.IP = msg.VTEP
		fdb.Family = syscall.AF_BRIDGE
		fdb.Flags = netlink.NTF_SELF
		fdb.Type = 0
		fdb.State = 0

		err = a.Netlinker.SetNeigh(fdb)
		if err != nil {
			a.Logger.Error("set-l2-forward-failed", err)
			continue
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
