package watcher

import (
	"net"
	"path/filepath"

	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o ../fakes/resolver.go --fake-name Resolver . resolver
type resolver interface {
	ResolveMisses(misses <-chan Neighbor, knownNeighbors chan<- Neighbor)
}

type Resolver struct {
	Logger lager.Logger
	Store  store.Store
}

func (d *Resolver) ResolveMisses(misses <-chan Neighbor, knownNeighbors chan<- Neighbor) {
	for msg := range misses {
		d.Logger.Info("sandbox-miss", lager.Data{
			"sandbox": msg.SandboxName,
			"dest_ip": msg.Neigh.IP,
			"msg":     msg,
		})

		containers, err := d.Store.All()
		if err != nil {
			d.Logger.Error("store-retrieval-failed", err)
			continue
		}

		found := false
		for _, container := range containers {
			if container.SandboxName != filepath.Base(msg.SandboxName) {
				continue
			}

			if container.IP == msg.Neigh.IP.String() {
				mac, err := net.ParseMAC(container.MAC)
				if err != nil {
					d.Logger.Error("parse-mac-failed", err)
					break
				}

				msg.Neigh.HardwareAddr = mac
				msg.VTEP = net.ParseIP(container.HostIP)
				found = true
				break
			}
		}

		if !found {
			continue
		}

		d.Logger.Info("resolved", lager.Data{
			"msg":     msg,
			"hw_addr": msg.Neigh.HardwareAddr.String(),
		})

		knownNeighbors <- msg
	}

	close(knownNeighbors)

	return
}
