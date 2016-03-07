package watcher_test

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher/subscriber"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Watcher", func() {
	var (
		logger      *lagertest.TestLogger
		locker      *fakes.Locker
		sub         *fakes.Subscriber
		namespace   *fakes.Namespace
		missWatcher watcher.MissWatcher
	)

	BeforeEach(func() {
		sub = &fakes.Subscriber{}
		logger = lagertest.NewTestLogger("test")
		locker = &fakes.Locker{}
		namespace = &fakes.Namespace{}
		missWatcher = watcher.New(logger, sub, locker)
		namespace.ExecuteStub = func(callback func(ns *os.File) error) error {
			err := callback(nil)
			if err != nil {
				return fmt.Errorf("callback failed: %s", err)
			}
			return nil
		}
		namespace.NameReturns("some-namespace")
	})

	Describe("StartMonitor", func() {
		It("subscribes to sandbox l3 misses", func() {
			missWatcher.StartMonitor(namespace)

			Expect(sub.SubscribeCallCount()).To(Equal(1))
		})

		It("invokes the subscribe call from within the namespace", func() {
			namespace.ExecuteStub = func(callback func(ns *os.File) error) error {
				Expect(sub.SubscribeCallCount()).To(Equal(0))
				callback(nil)
				Expect(sub.SubscribeCallCount()).To(Equal(1))
				return nil
			}

			missWatcher.StartMonitor(namespace)
			Expect(namespace.ExecuteCallCount()).To(Equal(1))
		})

		It("forwards Miss messages to the Firehose", func() {
			sub.SubscribeStub = func(subChan chan<- *subscriber.Neigh, done <-chan struct{}) error {
				go func() {
					subChan <- &subscriber.Neigh{IP: net.ParseIP("1.2.3.4")}
				}()
				return nil
			}

			missWatcher.StartMonitor(namespace)

			Eventually(logger).Should(gbytes.Say("1.2.3.4.*%s", namespace.Name()))
		})

		Context("when the miss message doesn't have a destination IP", func() {
			It("does not forward it to the firehose", func() {
				sub.SubscribeStub = func(subChan chan<- *subscriber.Neigh, done <-chan struct{}) error {
					go func() {
						subChan <- &subscriber.Neigh{State: 42}
					}()
					return nil
				}

				missWatcher.StartMonitor(namespace)

				Consistently(logger).ShouldNot(gbytes.Say("test"))
			})
		})

		It("locks and unlocks to protect the map", func() {
			missWatcher.StartMonitor(namespace)
			Expect(locker.LockCallCount()).To(Equal(1))
			Expect(locker.UnlockCallCount()).To(Equal(1))
		})

		Context("when subscribe returns an error", func() {
			It("returns the error", func() {
				sub.SubscribeReturns(errors.New("some subscribe error"))
				err := missWatcher.StartMonitor(namespace)
				Expect(err).To(MatchError("callback failed: subscribe in some-namespace: some subscribe error"))
			})
		})

		Context("when Execute fails", func() {
			It("returns the error", func() {
				namespace.ExecuteReturns(errors.New("boom"))
				Expect(missWatcher.StartMonitor(namespace)).To(MatchError("boom"))
			})
		})
	})

	Describe("StopMonitor", func() {
		var complete chan struct{}
		BeforeEach(func() {
			complete = make(chan struct{})
			sub.SubscribeStub = func(ch chan<- *subscriber.Neigh, done <-chan struct{}) error {
				go func() {
					<-done
					complete <- struct{}{}
				}()
				return nil
			}
			missWatcher.StartMonitor(namespace)
		})

		It("sends a done signal to the subscribed channel", func() {
			missWatcher.StopMonitor(namespace)

			Eventually(complete).Should(Receive())
		})

		It("locks and unlocks to protect the map", func() {
			missWatcher.StopMonitor(namespace)
			Eventually(complete).Should(Receive())

			Expect(locker.LockCallCount()).To(Equal(2))
			Expect(locker.UnlockCallCount()).To(Equal(2))
		})

		Context("when StopMonitor called many times", func() {
			It("returns a channel not found error", func() {
				Expect(missWatcher.StopMonitor(namespace)).To(Succeed())
				Eventually(complete).Should(Receive())

				Expect(missWatcher.StopMonitor(namespace)).To(MatchError("namespace some-namespace not monitored"))
			})
		})
	})

	Context("when StopMonitor called without subscription", func() {
		Context("when StartMonitor NEVER called", func() {
			It("returns a channel not found error", func() {
				Expect(missWatcher.StopMonitor(namespace)).To(MatchError("namespace some-namespace not monitored"))
			})
		})

	})
})
