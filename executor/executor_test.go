package executor_test

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	cmd_fakes "github.com/cloudfoundry-incubator/ducati-daemon/executor/commands/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Executor", func() {
	var (
		addressManager *fakes.AddressManager
		routeManager   *fakes.RouteManager
		linkFactory    *fakes.LinkFactory
		command        *cmd_fakes.Command
		ex             executor.Executor
	)

	BeforeEach(func() {
		addressManager = &fakes.AddressManager{}
		routeManager = &fakes.RouteManager{}
		linkFactory = &fakes.LinkFactory{}
		command = &cmd_fakes.Command{}

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

		Describe("AddressManager", func() {
			It("returns the address manager", func() {
				Expect(context.AddressManager()).To(Equal(addressManager))
			})
		})

		Describe("LinkFactory", func() {
			It("returns the link factory", func() {
				Expect(context.LinkFactory()).To(Equal(linkFactory))
			})
		})

		Describe("RouteManager", func() {
			It("returns the route manager", func() {
				Expect(context.RouteManager()).To(Equal(routeManager))
			})
		})
	})
})
