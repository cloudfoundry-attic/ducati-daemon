package watcher_test

import (
	"errors"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Resolver", func() {
	var (
		drainer               watcher.Resolver
		logger                *lagertest.TestLogger
		missesChannel         chan watcher.Neighbor
		fakeStore             *fakes.Store
		knownNeighborsChannel chan watcher.Neighbor
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		missesChannel = make(chan watcher.Neighbor)
		knownNeighborsChannel = make(chan watcher.Neighbor)
		fakeStore = &fakes.Store{}

		drainer = watcher.Resolver{
			Logger: logger,
			Store:  fakeStore,
		}

		go drainer.ResolveMisses(missesChannel, knownNeighborsChannel)
	})

	AfterEach(func() {
		Consistently(knownNeighborsChannel).ShouldNot(BeClosed())
		close(missesChannel)
		Eventually(knownNeighborsChannel).Should(BeClosed())
	})

	Describe("ResolveMisses", func() {
		var msg watcher.Neighbor

		BeforeEach(func() {
			msg = watcher.Neighbor{
				SandboxName: "some-sandbox-name",
				Neigh: watcher.Neigh{
					IP: net.ParseIP("192.168.1.2"),
				},
			}

			fakeStore.AllReturns([]models.Container{
				models.Container{
					IP: "1.2.3.4",
				},
				models.Container{
					IP:  "192.168.1.2",
					MAC: "ff:ff:ff:ff:ff:ff",
				},
			}, nil)
		})

		It("retrieves the MAC address associated with each incoming IP", func() {
			missesChannel <- msg

			Eventually(fakeStore.AllCallCount).Should(Equal(1))
		})

		It("puts a resolved neighbor message into the output channel", func() {
			missesChannel <- msg

			expectedMAC, _ := net.ParseMAC("ff:ff:ff:ff:ff:ff")
			Eventually(knownNeighborsChannel).Should(Receive(Equal(watcher.Neighbor{
				SandboxName: "some-sandbox-name",
				Neigh: watcher.Neigh{
					IP:           net.ParseIP("192.168.1.2"),
					HardwareAddr: expectedMAC,
				},
			})))
		})

		Context("when store fails", func() {
			BeforeEach(func() {
				fakeStore.AllReturns(nil, errors.New("banana"))
				missesChannel <- msg
			})

			It("logs the error", func() {
				Eventually(logger).Should(gbytes.Say("banana"))
			})

			It("does not propogate the message", func() {
				Consistently(knownNeighborsChannel).ShouldNot(Receive())
			})
		})

		Context("when the store results do not contain a match", func() {
			BeforeEach(func() {
				msg.Neigh.IP = net.ParseIP("10.12.13.14")
				missesChannel <- msg
			})

			It("does not process the message", func() {
				Consistently(knownNeighborsChannel).ShouldNot(Receive())
			})
		})

		Context("when fails to parseMAC of matching container", func() {
			BeforeEach(func() {
				fakeStore.AllReturns([]models.Container{
					models.Container{
						IP:  "192.168.1.2",
						MAC: "bad-mac",
					},
				}, nil)

				missesChannel <- msg
			})

			It("logs the error", func() {
				Eventually(logger).Should(gbytes.Say("parse-mac-failed.*bad-mac"))
			})

			It("does not propogate the message", func() {
				Consistently(knownNeighborsChannel).ShouldNot(Receive())
			})
		})
	})
})
