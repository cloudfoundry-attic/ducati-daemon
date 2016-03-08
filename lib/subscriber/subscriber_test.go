package subscriber_test

import (
	"errors"
	"net"
	"syscall"
	"time"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/nl"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/nl/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/subscriber"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/vishvananda/netlink"
)

var _ = Describe("Subscriber (real messages)", func() {
	var (
		neighChan    chan *watcher.Neigh
		doneChan     chan struct{}
		mySubscriber *subscriber.Subscriber
		logger       *lagertest.TestLogger
	)

	BeforeEach(func() {
		neighChan = make(chan *watcher.Neigh, 100)
		doneChan = make(chan struct{})
		logger = lagertest.NewTestLogger("test")

		mySubscriber = &subscriber.Subscriber{
			Netlinker: nl.Netlink,
			Logger:    logger,
		}
	})

	It("catches tcp connection misses", func() {
		err := mySubscriber.Subscribe(neighChan, doneChan)
		Expect(err).NotTo(HaveOccurred())

		_, err = net.Dial("tcp", "172.17.0.105:1234")
		Expect(err).To(HaveOccurred())

		Eventually(neighChan).Should(Receive())
	})
})

var _ = Describe("Subscriber (mock messages)", func() {
	var (
		fakeNetlinker *fakes.Netlinker
		fakeSocket    *fakes.NLSocket
		mySubscriber  *subscriber.Subscriber
		neighChan     chan *watcher.Neigh
		doneChan      chan struct{}
		logger        *lagertest.TestLogger
	)

	BeforeEach(func() {
		fakeNetlinker = &fakes.Netlinker{}
		fakeSocket = &fakes.NLSocket{}
		neighChan = make(chan *watcher.Neigh, 100)
		doneChan = make(chan struct{})
		logger = lagertest.NewTestLogger("test")

		mySubscriber = &subscriber.Subscriber{
			Netlinker: fakeNetlinker,
			Logger:    logger,
		}

		fakeNetlinker.SubscribeReturns(fakeSocket, nil)
		fakeSocket.ReceiveReturns([]syscall.NetlinkMessage{{Data: []byte("something")}}, nil)
		fakeNetlinker.NeighDeserializeReturns(&netlink.Neigh{}, nil)
	})

	It("faithfully represents the netlink Neighbor in the return type", func() {
		someMac, _ := net.ParseMAC("01:02:03:04:05:06")
		fakeNetlinker.NeighDeserializeReturns(&netlink.Neigh{
			LinkIndex:    1,
			Family:       2,
			State:        3,
			Type:         4,
			Flags:        5,
			IP:           net.ParseIP("1.2.3.4"),
			HardwareAddr: someMac,
		}, nil)

		err := mySubscriber.Subscribe(neighChan, doneChan)
		Expect(err).NotTo(HaveOccurred())

		Eventually(neighChan).Should(Receive(Equal(&watcher.Neigh{
			LinkIndex:    1,
			Family:       2,
			State:        3,
			Type:         4,
			Flags:        5,
			IP:           net.ParseIP("1.2.3.4"),
			HardwareAddr: someMac,
		})))
	})

	Context("when a message is sent on the done channel", func() {
		BeforeEach(func() {
			fakeSocket.ReceiveStub = func() ([]syscall.NetlinkMessage, error) {
				if fakeSocket.CloseCallCount() > 0 {
					return nil, errors.New("socket is closed!!!!")
				}
				time.Sleep(100 * time.Millisecond)
				return []syscall.NetlinkMessage{{Data: []byte("something")}}, nil
			}
		})
		It("closes the output channel", func() {
			err := mySubscriber.Subscribe(neighChan, doneChan)
			Expect(err).NotTo(HaveOccurred())

			Consistently(neighChan).ShouldNot(BeClosed())

			go func() {
				doneChan <- struct{}{}
			}()

			Eventually(neighChan).Should(BeClosed())
		})
	})

	Context("when netlink Subscribe fails", func() {
		BeforeEach(func() {
			fakeNetlinker.SubscribeReturns(nil, errors.New("some error"))
		})
		It("returns the error", func() {
			err := mySubscriber.Subscribe(neighChan, doneChan)
			Expect(err).To(MatchError("failed to acquire netlink socket: some error"))
		})
	})

	Context("when receive message fails", func() {
		BeforeEach(func() {
			fakeSocket.ReceiveReturns(nil, errors.New("some error"))
		})
		It("closes the output channel and logs the error", func() {
			err := mySubscriber.Subscribe(neighChan, doneChan)
			Expect(err).NotTo(HaveOccurred())

			Eventually(neighChan).Should(BeClosed())
			Expect(logger).To(gbytes.Say("socket receive.*some error"))
		})
	})

	Context("when neigh deserialize of message fails", func() {
		BeforeEach(func() {
			fakeNetlinker.NeighDeserializeReturns(nil, errors.New("some error"))
		})
		It("closes the output channel and logs the error", func() {
			err := mySubscriber.Subscribe(neighChan, doneChan)
			Expect(err).NotTo(HaveOccurred())

			Eventually(neighChan).Should(BeClosed())
			Eventually(logger).Should(gbytes.Say("neighbor deserialize.*some error"))
		})
	})
})
