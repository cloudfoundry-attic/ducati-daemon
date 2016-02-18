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
			link = &netlink.Veth{
				LinkAttrs: netlink.LinkAttrs{
					Index: 999,
				},
			}
			netlinker.LinkByNameReturns(link, nil)

			gateway = net.ParseIP("172.16.1.1")
			network = &net.IPNet{
				IP:   gateway,
				Mask: net.CIDRMask(24, 32),
			}
		})

		It("finds the link by name", func() {
			err := routeManager.AddRoute("my-link", network, gateway)
			Expect(err).NotTo(HaveOccurred())

			Expect(netlinker.LinkByNameCallCount()).To(Equal(1))
			Expect(netlinker.LinkByNameArgsForCall(0)).To(Equal("my-link"))
		})

		Context("when finding the link fails", func() {
			BeforeEach(func() {
				netlinker.LinkByNameReturns(nil, errors.New("no link for you"))
			})

			It("returns a meaningful error", func() {
				err := routeManager.AddRoute("my-link", network, gateway)
				Expect(err).To(MatchError("link by name failed: no link for you"))
			})
		})

		It("adds a route", func() {
			err := routeManager.AddRoute("my-link", network, gateway)
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
				netlinker.RouteAddReturns(errors.New("welp"))
			})

			It("returns the error", func() {
				err := routeManager.AddRoute("my-link", network, gateway)
				Expect(err).To(MatchError("route add failed: welp"))
			})
		})
	})
})
