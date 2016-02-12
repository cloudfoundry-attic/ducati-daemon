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
	var (
		netlinker      *fakes.Netlinker
		addressManager *ip.AddressManager
	)

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
			var err error
			link = &netlink.Veth{}
			_, address, err = net.ParseCIDR("192.168.1.1/24")
			Expect(err).NotTo(HaveOccurred())
		})

		It("adds an address to the link", func() {
			err := addressManager.AddAddress(link, address)
			Expect(err).NotTo(HaveOccurred())

			Expect(netlinker.AddrAddCallCount()).To(Equal(1))
			actualLink, netlinkAddr := netlinker.AddrAddArgsForCall(0)

			Expect(actualLink).To(Equal(link))
			Expect(netlinkAddr.IPNet).To(Equal(address))
		})

		Context("when adding the addres fails", func() {
			BeforeEach(func() {
				netlinker.AddrAddReturns(errors.New("adding address failed"))
			})

			It("returns the error", func() {
				err := addressManager.AddAddress(link, address)
				Expect(err).To(MatchError("adding address failed"))
			})
		})
	})
})
