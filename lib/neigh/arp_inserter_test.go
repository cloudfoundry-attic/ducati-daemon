package neigh_test

import (
	"errors"
	"net"
	"os"
	"syscall"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/neigh"
	nl_fakes "github.com/cloudfoundry-incubator/ducati-daemon/lib/nl/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("ARPInserter", func() {
	var (
		inserter     *neigh.ARPInserter
		ns           *fakes.Namespace
		netlinker    *nl_fakes.Netlinker
		vxlanLink    *netlink.Vxlan
		logger       *lagertest.TestLogger
		threadLocker *fakes.OSThreadLocker
	)

	BeforeEach(func() {
		ns = &fakes.Namespace{}
		netlinker = &nl_fakes.Netlinker{}
		logger = lagertest.NewTestLogger("test")
		threadLocker = &fakes.OSThreadLocker{}
		vxlanLink = &netlink.Vxlan{
			LinkAttrs: netlink.LinkAttrs{
				Index: 9876,
			},
		}

		ns.ExecuteStub = func(callback func(ns *os.File) error) error {
			return callback(nil)
		}

		netlinker.LinkByNameStub = func(linkName string) (netlink.Link, error) {
			return vxlanLink, nil
		}

		inserter = &neigh.ARPInserter{
			Logger:         logger,
			Netlinker:      netlinker,
			OSThreadLocker: threadLocker,
		}
	})

	Describe("HandleResolvedNeighbors", func() {
		var (
			neigh    watcher.Neigh
			neighbor watcher.Neighbor
			resolved chan watcher.Neighbor
			ready    chan error
		)

		BeforeEach(func() {
			mac, err := net.ParseMAC("01:02:03:04:05:06")
			Expect(err).NotTo(HaveOccurred())

			neigh = watcher.Neigh{
				LinkIndex:    99,
				Family:       22,
				State:        11,
				Type:         31,
				Flags:        17,
				IP:           net.ParseIP("1.2.3.4"),
				HardwareAddr: mac,
			}

			neighbor = watcher.Neighbor{
				SandboxName: "some-sandbox",
				VTEP:        net.ParseIP("10.11.12.13"),
				Neigh:       neigh,
			}

			resolved = make(chan watcher.Neighbor, 3)
			resolved <- neighbor

			ready = make(chan error, 1)
		})

		JustBeforeEach(func() {
			close(resolved)
		})

		AfterEach(func() {
			Eventually(ready).Should(BeClosed())
		})

		It("finds the vxlan device in the sandbox namespace", func() {
			ns.ExecuteStub = func(callback func(_ *os.File) error) error {
				Expect(netlinker.LinkByNameCallCount()).To(Equal(0))
				callback(nil)
				Expect(netlinker.LinkByNameCallCount()).To(Equal(1))
				return nil
			}

			inserter.HandleResolvedNeighbors(ready, ns, "some-vxlan-name", resolved)
			Eventually(ready).Should(BeClosed())

			Expect(ns.ExecuteCallCount()).To(Equal(1))
			Expect(netlinker.LinkByNameArgsForCall(0)).To(Equal("some-vxlan-name"))
		})

		Context("when the vxlan device cannot be found", func() {
			BeforeEach(func() {
				netlinker.LinkByNameReturns(nil, errors.New("boom-boom"))
			})

			It("returns a meaningful error", func() {
				inserter.HandleResolvedNeighbors(ready, ns, "some-vxlan-name", resolved)
				Eventually(ready).Should(Receive(MatchError(`namespace execute failed: find link "some-vxlan-name": boom-boom`)))
			})
		})

		It("sets a neighbor entry in the sandbox namespace", func() {
			ns.ExecuteStub = func(callback func(ns *os.File) error) error {
				Expect(netlinker.SetNeighCallCount()).To(Equal(0))
				callback(nil)
				Expect(netlinker.SetNeighCallCount()).To(Equal(2))
				return nil
			}

			inserter.HandleResolvedNeighbors(ready, ns, "some-vxlan-name", resolved)
			Eventually(ready).Should(BeClosed())

			Expect(ns.ExecuteCallCount()).To(Equal(1))
			Expect(netlinker.SetNeighArgsForCall(0)).To(Equal(&netlink.Neigh{
				LinkIndex:    neigh.LinkIndex,
				Family:       neigh.Family,
				State:        netlink.NUD_REACHABLE,
				Type:         neigh.Type,
				Flags:        neigh.Flags,
				IP:           neigh.IP,
				HardwareAddr: neigh.HardwareAddr,
			}))
		})

		It("sets a forwarding database entry in the the namespace", func() {
			ns.ExecuteStub = func(callback func(_ *os.File) error) error {
				Expect(netlinker.SetNeighCallCount()).To(Equal(0))
				callback(nil)
				Expect(netlinker.SetNeighCallCount()).To(Equal(2))
				return nil
			}

			inserter.HandleResolvedNeighbors(ready, ns, "some-vxlan-name", resolved)
			Eventually(ready).Should(BeClosed())

			Expect(ns.ExecuteCallCount()).To(Equal(1))
			Expect(netlinker.SetNeighArgsForCall(1)).To(Equal(&netlink.Neigh{
				LinkIndex:    9876,
				HardwareAddr: neigh.HardwareAddr,
				Family:       syscall.AF_BRIDGE,
				State:        0,
				Type:         0,
				Flags:        netlink.NTF_SELF,
				IP:           neighbor.VTEP,
			}))
		})

		It("locks and unlocks the OS thread", func() {
			ns.ExecuteStub = func(callback func(ns *os.File) error) error {
				callback(nil)
				Expect(threadLocker.LockOSThreadCallCount()).To(Equal(1))
				Expect(threadLocker.UnlockOSThreadCallCount()).To(Equal(0))
				return nil
			}

			inserter.HandleResolvedNeighbors(ready, ns, "some-vxlan-name", resolved)
			Eventually(ready).Should(BeClosed())

			Expect(ns.ExecuteCallCount()).To(Equal(1))
			Expect(threadLocker.LockOSThreadCallCount()).To(Equal(1))
			Expect(threadLocker.UnlockOSThreadCallCount()).To(Equal(1))
		})

		Context("when executing in namespace fails", func() {
			BeforeEach(func() {
				ns.ExecuteReturns(errors.New("peppers"))
			})

			It("returns the error", func() {
				inserter.HandleResolvedNeighbors(ready, ns, "some-vxlan-name", resolved)
				Eventually(ready).Should(Receive(MatchError("namespace execute failed: peppers")))
			})
		})

		Context("when setting a neighbor entry fails", func() {
			BeforeEach(func() {
				netlinker.SetNeighStub = func(n *netlink.Neigh) error {
					if netlinker.SetNeighCallCount() == 1 {
						return errors.New("go huskies")
					}
					return nil
				}

				resolved <- watcher.Neighbor{}
			})

			It("logs the error and continues in the loop", func() {
				inserter.HandleResolvedNeighbors(ready, ns, "some-vxlan-name", resolved)
				Eventually(ready).Should(BeClosed())

				Eventually(logger).Should(gbytes.Say("set-l3-neighbor-failed.*huskies"))
				Expect(netlinker.SetNeighCallCount()).To(Equal(3))
			})
		})

		Context("when setting the forwarding entry fails", func() {
			BeforeEach(func() {
				netlinker.SetNeighStub = func(n *netlink.Neigh) error {
					if netlinker.SetNeighCallCount() == 2 {
						return errors.New("fail-on-two")
					}
					return nil
				}
				resolved <- watcher.Neighbor{}
			})

			It("logs the error and continues", func() {
				inserter.HandleResolvedNeighbors(ready, ns, "some-vxlan-name", resolved)
				Eventually(ready).Should(BeClosed())

				Eventually(logger).Should(gbytes.Say("set-l2-forward-failed.*fail-on-two"))
				Expect(netlinker.SetNeighCallCount()).To(Equal(4))
			})
		})
	})
})
