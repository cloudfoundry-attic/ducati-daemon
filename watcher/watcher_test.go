package watcher_test

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Watcher", func() {
	var (
		logger        *lagertest.TestLogger
		locker        *fakes.Locker
		sub           *fakes.Subscriber
		ns            *fakes.Namespace
		vxlanLinkName string
		resolver      *fakes.Resolver
		missWatcher   watcher.MissWatcher
		arpInserter   *fakes.ARPInserter
	)

	BeforeEach(func() {
		sub = &fakes.Subscriber{}
		logger = lagertest.NewTestLogger("test")
		locker = &fakes.Locker{}
		ns = &fakes.Namespace{}
		vxlanLinkName = "some-vxlan-name"
		resolver = &fakes.Resolver{}

		arpInserter = &fakes.ARPInserter{}
		arpInserter.HandleResolvedNeighborsStub = func(ready chan error, _ namespace.Executor, _ string, _ <-chan watcher.Neighbor) {
			close(ready)
		}

		missWatcher = watcher.New(sub, locker, resolver, arpInserter)

		ns.ExecuteStub = func(callback func(ns *os.File) error) error {
			err := callback(nil)
			if err != nil {
				return fmt.Errorf("callback failed: %s", err)
			}
			return nil
		}
		ns.NameReturns("some-namespace")
	})

	Describe("StartMonitor", func() {
		It("subscribes to sandbox l3 misses", func() {
			missWatcher.StartMonitor(ns, vxlanLinkName)

			Expect(sub.SubscribeCallCount()).To(Equal(1))
		})

		It("invokes the subscribe call from within the namespace", func() {
			ns.ExecuteStub = func(callback func(ns *os.File) error) error {
				Expect(sub.SubscribeCallCount()).To(Equal(0))
				callback(nil)
				Expect(sub.SubscribeCallCount()).To(Equal(1))
				return nil
			}

			missWatcher.StartMonitor(ns, vxlanLinkName)
			Expect(ns.ExecuteCallCount()).To(Equal(1))
		})

		It("forwards Neighbor messages to the resolver, running in a separate goroutine", func() {
			sub.SubscribeStub = func(subChan chan<- *watcher.Neigh, done <-chan struct{}) error {
				go func() {
					subChan <- &watcher.Neigh{IP: net.ParseIP("1.2.3.4")}
				}()
				return nil
			}

			missWatcher.StartMonitor(ns, vxlanLinkName)

			Eventually(resolver.ResolveMissesCallCount).Should(Equal(1))
			misses, resolved := resolver.ResolveMissesArgsForCall(0)
			Expect(resolved).NotTo(BeNil())

			Eventually(misses).Should(Receive())
		})

		It("starts the APRInserter and waits on the ready channel", func() {
			arpInserter.HandleResolvedNeighborsStub = nil

			result := make(chan error, 1)
			go func() { result <- missWatcher.StartMonitor(ns, vxlanLinkName) }()

			Eventually(arpInserter.HandleResolvedNeighborsCallCount).Should(Equal(1))

			ready, sboxNS, vxlanName, inserterResolved := arpInserter.HandleResolvedNeighborsArgsForCall(0)
			Consistently(ready).ShouldNot(BeClosed())
			Expect(sboxNS).To(Equal(ns))
			Expect(vxlanName).To(Equal("some-vxlan-name"))
			Consistently(inserterResolved).ShouldNot(BeClosed())

			close(ready)
			Eventually(result).Should(Receive(BeNil()))
		})

		It("forwards resolved misses to the arp inserter", func() {
			missWatcher.StartMonitor(ns, vxlanLinkName)

			Eventually(arpInserter.HandleResolvedNeighborsCallCount).Should(Equal(1))
			Eventually(resolver.ResolveMissesCallCount).Should(Equal(1))

			_, resolverResolved := resolver.ResolveMissesArgsForCall(0)
			_, _, _, inserterResolved := arpInserter.HandleResolvedNeighborsArgsForCall(0)

			go func() {
				resolverResolved <- watcher.Neighbor{SandboxName: "thingy"}
			}()

			var neigh watcher.Neighbor
			Eventually(inserterResolved).Should(Receive(&neigh))
			Expect(neigh).To(Equal(watcher.Neighbor{SandboxName: "thingy"}))
		})

		Context("when SyncHandleResolvedNeighbors fails", func() {
			BeforeEach(func() {
				arpInserter.HandleResolvedNeighborsStub = func(ready chan error, _ namespace.Executor, _ string, _ <-chan watcher.Neighbor) {
					ready <- errors.New("zuccini")
				}
			})

			It("returns the error", func() {
				err := missWatcher.StartMonitor(ns, vxlanLinkName)
				Expect(err).To(MatchError("arp inserter failed: handle resolved: zuccini"))
			})

			It("subscriber does not get called", func() {
				missWatcher.StartMonitor(ns, vxlanLinkName)
				Consistently(sub.SubscribeCallCount).Should(Equal(0))
			})
		})

		Context("when the miss message doesn't have a destination IP", func() {
			It("does not forward it to the firehose", func() {
				sub.SubscribeStub = func(subChan chan<- *watcher.Neigh, done <-chan struct{}) error {
					go func() {
						subChan <- &watcher.Neigh{State: 42}
					}()
					return nil
				}

				missWatcher.StartMonitor(ns, vxlanLinkName)

				Consistently(logger).ShouldNot(gbytes.Say("test"))
			})
		})

		It("locks and unlocks to protect the map", func() {
			missWatcher.StartMonitor(ns, vxlanLinkName)
			Expect(locker.LockCallCount()).To(Equal(1))
			Expect(locker.UnlockCallCount()).To(Equal(1))
		})

		Context("when subscribe returns an error", func() {
			It("returns the error", func() {
				sub.SubscribeReturns(errors.New("some subscribe error"))
				err := missWatcher.StartMonitor(ns, vxlanLinkName)
				Expect(err).To(MatchError("callback failed: subscribe in some-namespace: some subscribe error"))
			})
		})

		Context("when Execute fails", func() {
			It("returns the error", func() {
				ns.ExecuteReturns(errors.New("boom"))
				Expect(missWatcher.StartMonitor(ns, vxlanLinkName)).To(MatchError("boom"))
			})
		})
	})

	Describe("StopMonitor", func() {
		var complete chan struct{}
		BeforeEach(func() {
			complete = make(chan struct{})
			sub.SubscribeStub = func(ch chan<- *watcher.Neigh, done <-chan struct{}) error {
				go func() {
					<-done
					complete <- struct{}{}
				}()
				return nil
			}
			missWatcher.StartMonitor(ns, vxlanLinkName)
		})

		It("sends a done signal to the subscribed channel", func() {
			missWatcher.StopMonitor(ns)

			Eventually(complete).Should(Receive())
		})

		It("locks and unlocks to protect the map", func() {
			missWatcher.StopMonitor(ns)
			Eventually(complete).Should(Receive())

			Expect(locker.LockCallCount()).To(Equal(2))
			Expect(locker.UnlockCallCount()).To(Equal(2))
		})

		Context("when StopMonitor called many times", func() {
			It("returns a channel not found error", func() {
				Expect(missWatcher.StopMonitor(ns)).To(Succeed())
				Eventually(complete).Should(Receive())

				Expect(missWatcher.StopMonitor(ns)).To(MatchError("namespace some-namespace not monitored"))
			})
		})
	})

	Context("when StopMonitor called without subscription", func() {
		Context("when StartMonitor NEVER called", func() {
			It("returns a channel not found error", func() {
				Expect(missWatcher.StopMonitor(ns)).To(MatchError("namespace some-namespace not monitored"))
			})
		})

	})
})
