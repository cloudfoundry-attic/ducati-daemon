package watcher

import (
	"fmt"
	"net"
	"os"
	"sync"

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
	Subscribe(ch chan<- *Neigh, done <-chan struct{}) error
}

type Namespace interface {
	Execute(func(*os.File) error) error
	Name() string
}

//go:generate counterfeiter -o ../fakes/watcher.go --fake-name MissWatcher . MissWatcher
type MissWatcher interface {
	StartMonitor(Namespace) error
	StopMonitor(Namespace) error
}

func New(logger lager.Logger, subscriber sub, locker sync.Locker) MissWatcher {
	firehose := make(chan Miss)
	w := &missWatcher{
		Logger:     logger,
		Subscriber: subscriber,
		DoneChans:  make(map[string]chan struct{}),
		Locker:     locker,
		Firehose:   firehose,
		Drainer: &Drainer{
			Logger:   logger,
			Firehose: firehose,
		},
	}
	go w.Drainer.Drain()
	return w
}

type missWatcher struct {
	Logger     lager.Logger
	Subscriber sub
	DoneChans  map[string]chan struct{}
	Locker     sync.Locker
	Firehose   chan Miss
	Drainer    *Drainer
}

type Miss struct {
	SandboxName string
	DestIP      net.IP
}

func (w *missWatcher) StartMonitor(ns Namespace) error {
	subChan := make(chan *Neigh)

	doneChan := make(chan struct{})

	w.Locker.Lock()
	w.DoneChans[ns.Name()] = doneChan
	w.Locker.Unlock()

	err := ns.Execute(func(f *os.File) error {
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

			miss := Miss{
				SandboxName: ns.Name(),
				DestIP:      neigh.IP,
			}

			w.Firehose <- miss
		}
	}()

	return nil
}

func (w *missWatcher) StopMonitor(ns Namespace) error {
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
