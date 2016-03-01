package executor_test

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Executor", func() {
	var (
		addressManager *fakes.AddressManager
		routeManager   *fakes.RouteManager
		linkFactory    *fakes.LinkFactory
		command        *fakes.Command
		ex             executor.Executor
	)

	BeforeEach(func() {
		addressManager = &fakes.AddressManager{}
		routeManager = &fakes.RouteManager{}
		linkFactory = &fakes.LinkFactory{}
		command = &fakes.Command{}

		ex = executor.New(addressManager, routeManager, linkFactory)
	})

	It("executes a command with a context", func() {
		err := ex.Execute(command)
		Expect(err).NotTo(HaveOccurred())

		Expect(command.ExecuteCallCount()).To(Equal(1))
	})

	Describe("Context", func() {
		var context commands.Context

		BeforeEach(func() {
			c, ok := ex.(commands.Context)
			Expect(ok).To(BeTrue())
			context = c
		})

		Describe("AddressAdder", func() {
			It("returns the address manager", func() {
				Expect(context.AddressAdder()).To(Equal(addressManager))
			})
		})

		Describe("RouteAdder", func() {
			It("returns the address manager", func() {
				Expect(context.RouteAdder()).To(Equal(routeManager))
			})
		})

		Describe("BridgeFactory", func() {
			It("returns the link factory", func() {
				Expect(context.BridgeFactory()).To(Equal(linkFactory))
			})
		})

		Describe("HardwareAddresser", func() {
			It("returns the link factory", func() {
				Expect(context.HardwareAddresser()).To(Equal(linkFactory))
			})
		})

		Describe("MasterSetter", func() {
			It("returns the link factory", func() {
				Expect(context.MasterSetter()).To(Equal(linkFactory))
			})
		})

		Describe("SetNamespacer", func() {
			It("returns the link factory", func() {
				Expect(context.SetNamespacer()).To(Equal(linkFactory))
			})
		})

		Describe("SetUpper", func() {
			It("returns the link factory", func() {
				Expect(context.SetUpper()).To(Equal(linkFactory))
			})
		})

		Describe("VethFactory", func() {
			It("returns the link factory", func() {
				Expect(context.VethFactory()).To(Equal(linkFactory))
			})
		})

		Describe("VxlanFactory", func() {
			It("returns the link factory", func() {
				Expect(context.VxlanFactory()).To(Equal(linkFactory))
			})
		})

		Describe("LinkDeletor", func() {
			It("returns the link deletor", func() {
				Expect(context.LinkDeletor()).To(Equal(linkFactory))
			})
		})
	})
})
