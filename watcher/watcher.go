package watcher

import (
	"fmt"
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
	return &missWatcher{
		Logger:     logger,
		Subscriber: subscriber,
		DoneChans:  make(map[string]chan struct{}),
		Locker:     locker,
	}
}

type missWatcher struct {
	Logger     lager.Logger
	Subscriber sub
	DoneChans  map[string]chan struct{}
	Locker     sync.Locker
}

func (w *missWatcher) StartMonitor(ns Namespace) error {
	ch := make(chan<- *subscriber.Neigh, 100)

	doneChan := make(chan struct{})

	w.Locker.Lock()
	w.DoneChans[ns.Name()] = doneChan
	w.Locker.Unlock()

	err := ns.Execute(func(f *os.File) error {
		err := w.Subscriber.Subscribe(ch, doneChan)
		if err != nil {
			return fmt.Errorf("subscribe in %s: %s", ns.Name(), err)
		}
		return nil
	})
	if err != nil {
		return err
	}

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
