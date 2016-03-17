package watcher

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
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
	Subscribe(ch chan<- *Neigh, done <-chan struct{}) error
}

//go:generate counterfeiter -o ../fakes/watcher.go --fake-name MissWatcher . MissWatcher
type MissWatcher interface {
	StartMonitor(namespace.Executor) error
	StopMonitor(namespace.Executor) error
}

//go:generate counterfeiter -o ../fakes/arp_inserter.go --fake-name ARPInserter . arpInserter
type arpInserter interface {
	HandleResolvedNeighbors(ns namespace.Executor, resolvedNeighbors <-chan Neighbor) error
}

func New(subscriber sub, locker sync.Locker, resolver resolver, arpInserter arpInserter) MissWatcher {
	w := &missWatcher{
		Subscriber:  subscriber,
		DoneChans:   make(map[string]chan struct{}),
		Locker:      locker,
		Resolver:    resolver,
		ARPInserter: arpInserter,
	}

	return w
}

type missWatcher struct {
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

func (w *missWatcher) StartMonitor(ns namespace.Executor) error {
	subChan := make(chan *Neigh)

	unresolvedMisses := make(chan Neighbor)
	resolvedNeighbors := make(chan Neighbor)

	doneChan := make(chan struct{})

	w.Locker.Lock()
	w.DoneChans[ns.Name()] = doneChan
	w.Locker.Unlock()

	err := w.ARPInserter.HandleResolvedNeighbors(ns, resolvedNeighbors)
	if err != nil {
		return fmt.Errorf("arp inserter failed: %s", err)
	}

	err = ns.Execute(func(f *os.File) error {
		err := w.Subscriber.Subscribe(subChan, doneChan)
		if err != nil {
			return fmt.Errorf("subscribe in %s: %s", ns.Name(), err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	go func() {
		for neigh := range subChan {
			if neigh.IP == nil {
				continue
			}

			unresolvedMisses <- Neighbor{
				SandboxName: ns.Name(),
				Neigh:       *neigh,
			}
		}
	}()

	go w.Resolver.ResolveMisses(unresolvedMisses, resolvedNeighbors)

	return nil
}

func (w *missWatcher) StopMonitor(ns namespace.Executor) error {
	w.Locker.Lock()
	defer w.Locker.Unlock()

	doneChan, ok := w.DoneChans[ns.Name()]
	if !ok {
		return fmt.Errorf("namespace %s not monitored", ns.Name())
	}

	delete(w.DoneChans, ns.Name())

	doneChan <- struct{}{}

	return nil
}
