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

func (a *ARPInserter) HandleResolvedNeighbors(ready chan error, ns namespace.Executor, vxlanDeviceName string, resolvedChan <-chan watcher.Neighbor) {
	a.OSThreadLocker.LockOSThread()
	defer a.OSThreadLocker.UnlockOSThread()

	err := ns.Execute(func(f *os.File) error {
		vxlanLink, err := a.Netlinker.LinkByName(vxlanDeviceName)
		if err != nil {
			return fmt.Errorf("find link %q: %s", vxlanDeviceName, err)
		}

		close(ready)

		a.addNeighbors(vxlanLink.Attrs().Index, resolvedChan)
		return nil
	})

	if err != nil {
		ready <- fmt.Errorf("namespace execute failed: %s", err)
		close(ready)
	}
}

func (a *ARPInserter) addNeighbors(vxlanLinkIndex int, resolvedChan <-chan watcher.Neighbor) {
	for msg := range resolvedChan {
		neigh := reverseConvert(msg.Neigh)
		if neigh.State == netlink.NUD_STALE {
			neigh.State = netlink.NUD_REACHABLE
		}

		err := a.Netlinker.SetNeigh(neigh)
		if err != nil {
			a.Logger.Error("set-l3-neighbor-failed", err)
			continue
		}
		a.Logger.Info("inserted-neigh", lager.Data{
			"neigh": neigh.String(),
		})

		fdb := &netlink.Neigh{
			LinkIndex:    vxlanLinkIndex,
			HardwareAddr: neigh.HardwareAddr,
			IP:           msg.VTEP,
			Family:       syscall.AF_BRIDGE,
			Flags:        netlink.NTF_SELF,
			State:        netlink.NUD_REACHABLE,
		}

		err = a.Netlinker.SetNeigh(fdb)
		if err != nil {
			a.Logger.Error("set-l2-forward-failed", err)
			continue
		}

		a.Logger.Info("inserted-fdb", lager.Data{
			"fdb":     fdb,
			"hw_addr": neigh.HardwareAddr.String(),
		})
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
