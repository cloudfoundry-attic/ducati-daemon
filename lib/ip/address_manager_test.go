package ip_test

import (
	"errors"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/ip"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/nl/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
)

var _ = Describe("AddressManager", func() {
	var netlinker *fakes.Netlinker
	var addressManager *ip.AddressManager

	BeforeEach(func() {
		netlinker = &fakes.Netlinker{}
		addressManager = &ip.AddressManager{
			Netlinker: netlinker,
		}
	})

	Describe("AddAddress", func() {
		var (
			link    netlink.Link
			address *net.IPNet
		)

		BeforeEach(func() {
			address = &net.IPNet{
				IP:   net.ParseIP("192.168.1.1"),
				Mask: net.CIDRMask(24, 32),
			}

			link = &netlink.Veth{}
			netlinker.LinkByNameReturns(link, nil)
		})

		It("finds the link by name", func() {
			err := addressManager.AddAddress("my-link", address)
			Expect(err).NotTo(HaveOccurred())

			Expect(netlinker.LinkByNameCallCount()).To(Equal(1))
			Expect(netlinker.LinkByNameArgsForCall(0)).To(Equal("my-link"))
		})

		Context("when finding the link fails", func() {
			BeforeEach(func() {
				netlinker.LinkByNameReturns(nil, errors.New("no link for you"))
			})

			It("returns a meaningful error", func() {
				err := addressManager.AddAddress("my-link", address)
				Expect(err).To(MatchError("link by name failed: no link for you"))
			})
		})

		It("adds an address to the link", func() {
			err := addressManager.AddAddress("my-link", address)
			Expect(err).NotTo(HaveOccurred())

			Expect(netlinker.AddrAddCallCount()).To(Equal(1))
			actualLink, netlinkAddr := netlinker.AddrAddArgsForCall(0)

			Expect(actualLink).To(Equal(link))
			Expect(netlinkAddr.IPNet).To(Equal(address))
		})

		Context("when adding the addres fails", func() {
			BeforeEach(func() {
				netlinker.AddrAddReturns(errors.New("welp"))
			})

			It("returns the error", func() {
				err := addressManager.AddAddress("my-link", address)
				Expect(err).To(MatchError("address add failed: welp"))
			})
		})
	})
})
