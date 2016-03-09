package commands_test

import (
	"errors"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddRoute", func() {
	var (
		routeAdder *fakes.RouteAdder
		context    *fakes.Context
		addRoute   commands.AddRoute
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		routeAdder = &fakes.RouteAdder{}
		context.RouteAdderReturns(routeAdder)

		addRoute = commands.AddRoute{
			Interface: "my-interface",
			Destination: net.IPNet{
				IP:   net.ParseIP("192.168.1.1"),
				Mask: net.CIDRMask(24, 32),
			},
			Gateway: net.ParseIP("192.168.1.4"),
		}
	})

	It("uses the route addder to add the route", func() {
		err := addRoute.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(routeAdder.AddRouteCallCount()).To(Equal(1))
		ifName, dest, gw := routeAdder.AddRouteArgsForCall(0)
		Expect(ifName).To(Equal("my-interface"))
		Expect(dest.String()).To(Equal("192.168.1.1/24"))
		Expect(gw.String()).To(Equal("192.168.1.4"))
	})

	Context("when the route adder fails", func() {
		BeforeEach(func() {
			routeAdder.AddRouteReturns(errors.New("no route for you"))
		})

		It("wraps and propogates the error", func() {
			err := addRoute.Execute(context)
			Expect(err).To(MatchError("add route: no route for you"))
		})
	})

	Describe("String", func() {
		It("describes itself", func() {
			Expect(addRoute.String()).To(Equal("ip route add dev my-interface 192.168.1.1/24 via 192.168.1.4"))
		})
	})
})
