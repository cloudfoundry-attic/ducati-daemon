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

var _ = Describe("RouteManager", func() {
	var (
		netlinker    *fakes.Netlinker
		routeManager *ip.RouteManager
	)

	BeforeEach(func() {
		netlinker = &fakes.Netlinker{}
		routeManager = &ip.RouteManager{
			Netlinker: netlinker,
		}
	})

	Describe("AddRoute", func() {
		var (
			link    netlink.Link
			network *net.IPNet
			gateway net.IP
		)

		BeforeEach(func() {
			var err error
			link = &netlink.Veth{
				LinkAttrs: netlink.LinkAttrs{
					Index: 999,
				},
			}
			gateway, network, err = net.ParseCIDR("172.16.1.1/24")
			Expect(err).NotTo(HaveOccurred())
		})

		It("adds a route", func() {
			err := routeManager.AddRoute(link, network, gateway)
			Expect(err).NotTo(HaveOccurred())

			Expect(netlinker.RouteAddCallCount()).To(Equal(1))
			route := netlinker.RouteAddArgsForCall(0)

			Expect(route).To(Equal(&netlink.Route{
				LinkIndex: 999,
				Scope:     netlink.SCOPE_UNIVERSE,
				Dst:       network,
				Gw:        gateway,
			}))
		})

		Context("when adding the route fails", func() {
			BeforeEach(func() {
				netlinker.RouteAddReturns(errors.New("route add failed"))
			})

			It("returns the error", func() {
				err := routeManager.AddRoute(link, network, gateway)
				Expect(err).To(MatchError("route add failed"))
			})
		})
	})
})
