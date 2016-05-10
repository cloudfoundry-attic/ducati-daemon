package neigh

import (
	"fmt"
	"os"
	"syscall"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
	"github.com/pivotal-golang/lager"
	"github.com/vishvananda/netlink"
)

type netlinker interface {
	LinkByName(name string) (netlink.Link, error)
	SetNeigh(*netlink.Neigh) error
}

type ARPInserter struct {
	Logger    lager.Logger
	Netlinker netlinker
}

func (a *ARPInserter) HandleResolvedNeighbors(ready chan error, ns namespace.Namespace, vxlanDeviceName string, resolvedChan <-chan watcher.Neighbor) {

	var vxlanLink netlink.Link
	err := ns.Execute(func(f *os.File) error {
		var err error
		vxlanLink, err = a.Netlinker.LinkByName(vxlanDeviceName)
		if err != nil {
			return fmt.Errorf("find link %q: %s", vxlanDeviceName, err)
		}
		return nil
	})
	if err != nil {
		ready <- fmt.Errorf("namespace execute failed: %s", err)
		close(ready)
		return
	}

	close(ready)

	a.addNeighbors(vxlanLink.Attrs().Index, ns, resolvedChan)
}

func (a *ARPInserter) addNeighbors(vxlanLinkIndex int, ns namespace.Namespace, resolvedChan <-chan watcher.Neighbor) {
	for msg := range resolvedChan {
		neigh := reverseConvert(msg.Neigh)
		neigh.State = netlink.NUD_REACHABLE

		fdb := &netlink.Neigh{
			LinkIndex:    vxlanLinkIndex,
			HardwareAddr: neigh.HardwareAddr,
			IP:           msg.VTEP,
			Family:       syscall.AF_BRIDGE,
			Flags:        netlink.NTF_SELF,
			State:        netlink.NUD_REACHABLE,
		}

		a.Logger.Info("adding-neigbor", lager.Data{
			"neigh":   neigh.String(),
			"fdb":     fdb,
			"hw_addr": neigh.HardwareAddr.String(),
		})

		err := ns.Execute(func(*os.File) error {
			err := a.Netlinker.SetNeigh(neigh)
			if err != nil {
				return fmt.Errorf("set L3 neighbor failed: %s", err)
			}

			err = a.Netlinker.SetNeigh(fdb)
			if err != nil {
				return fmt.Errorf("set L2 forward failed: %s", err)
			}

			return nil
		})
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
