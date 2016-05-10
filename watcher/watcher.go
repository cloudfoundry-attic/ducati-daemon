package watcher

import (
	"fmt"
	"net"
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/pivotal-golang/lager"
)

type Neigh struct {
	LinkIndex    int
	Family       int
	State        int
	Type         int
	Flags        int
	IP           net.IP
	HardwareAddr net.HardwareAddr
}

//go:generate counterfeiter -o ../fakes/subscriber.go --fake-name Subscriber . sub
type sub interface {
	Subscribe(ns namespace.Namespace, ch chan<- *Neigh, done <-chan struct{}) error
}

//go:generate counterfeiter -o ../fakes/watcher.go --fake-name MissWatcher . MissWatcher
type MissWatcher interface {
	StartMonitor(ns namespace.Namespace, vxlanLinkName string) error
	StopMonitor(ns namespace.Namespace) error
}

//go:generate counterfeiter -o ../fakes/arp_inserter.go --fake-name ARPInserter . arpInserter
type arpInserter interface {
	HandleResolvedNeighbors(ready chan error, ns namespace.Namespace, vxlanName string, resolvedNeighbors <-chan Neighbor)
}

func New(logger lager.Logger, subscriber sub, locker sync.Locker, resolver resolver, arpInserter arpInserter) MissWatcher {
	w := &missWatcher{
		Logger:      logger,
		Subscriber:  subscriber,
		DoneChans:   make(map[string]chan struct{}),
		Locker:      locker,
		Resolver:    resolver,
		ARPInserter: arpInserter,
	}

	return w
}

type missWatcher struct {
	Logger      lager.Logger
	Subscriber  sub
	DoneChans   map[string]chan struct{}
	Locker      sync.Locker
	Firehose    chan Neighbor
	ARPInserter arpInserter
	Resolver    resolver
}

type Neighbor struct {
	SandboxName string
	VTEP        net.IP
	Neigh       Neigh
}

func (w *missWatcher) StartMonitor(ns namespace.Namespace, vxlanName string) error {
	logger := w.Logger.Session("start-monitor", lager.Data{"namespace": ns})
	logger.Info("called")
	defer logger.Info("complete")

	subChan := make(chan *Neigh)
	unresolvedMisses := make(chan Neighbor)
	resolvedNeighbors := make(chan Neighbor)

	doneChan := make(chan struct{})

	w.Locker.Lock()
	w.DoneChans[ns.Name()] = doneChan
	w.Locker.Unlock()

	err := w.startARPInserter(ns, vxlanName, resolvedNeighbors)
	if err != nil {
		return fmt.Errorf("arp inserter failed: %s", err)
	}

	err = w.Subscriber.Subscribe(ns, subChan, doneChan)
	if err != nil {
		return fmt.Errorf("subscribe in %s: %s", ns.Name(), err)
	}

	go func() {
		logger := logger.Session("forward-neighbor-messages")
		logger.Info("starting")
		for neigh := range subChan {
			unresolvedMisses <- Neighbor{
				SandboxName: ns.Name(),
				Neigh:       *neigh,
			}
		}
		logger.Info("complete")
	}()

	go w.Resolver.ResolveMisses(unresolvedMisses, resolvedNeighbors)

	return nil
}

func (w *missWatcher) StopMonitor(ns namespace.Namespace) error {
	w.Locker.Lock()
	defer w.Locker.Unlock()

	logger := w.Logger.Session("stop-monitor", lager.Data{"namespace": ns})
	logger.Info("called")
	defer logger.Info("complete")

	doneChan, ok := w.DoneChans[ns.Name()]
	if !ok {
		err := fmt.Errorf("namespace %s not monitored", ns.Name())
		logger.Error("done-channel-missing", err)
		return err
	}

	delete(w.DoneChans, ns.Name())
	close(doneChan)

	return nil
}

func (w *missWatcher) startARPInserter(ns namespace.Namespace, vxlanDeviceName string, resolvedChan <-chan Neighbor) error {
	ready := make(chan error)

	go w.ARPInserter.HandleResolvedNeighbors(ready, ns, vxlanDeviceName, resolvedChan)

	err := <-ready
	if err != nil {
		return fmt.Errorf("handle resolved: %s", err)
	}

	return nil
}
