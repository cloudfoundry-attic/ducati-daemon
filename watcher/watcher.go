package watcher

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/cloudfoundry-incubator/ducati-daemon/watcher/subscriber"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o ../fakes/subscriber.go --fake-name Subscriber . sub
type sub interface {
	Subscribe(ch chan<- *subscriber.Neigh, done <-chan struct{}) error
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
	w := &missWatcher{
		Logger:     logger,
		Subscriber: subscriber,
		DoneChans:  make(map[string]chan struct{}),
		Locker:     locker,
		Firehose:   make(chan Miss),
	}
	go w.DrainFirehose()
	return w
}

type missWatcher struct {
	Logger     lager.Logger
	Subscriber sub
	DoneChans  map[string]chan struct{}
	Locker     sync.Locker
	Firehose   chan Miss
}

type Miss struct {
	SandboxName string
	DestIP      net.IP
}

func (w *missWatcher) DrainFirehose() {
	for {
		msg := <-w.Firehose
		w.Logger.Info("sandbox-miss", lager.Data{
			"sandbox": msg.SandboxName,
			"dest_ip": msg.DestIP,
		})
	}
}

func (w *missWatcher) StartMonitor(ns Namespace) error {
	subChan := make(chan *subscriber.Neigh)

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
