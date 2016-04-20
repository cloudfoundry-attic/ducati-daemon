package executor_test

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Executor", func() {
	var (
		addressManager             *fakes.AddressManager
		routeManager               *fakes.RouteManager
		linkFactory                *fakes.LinkFactory
		sandboxNamespaceRepository *fakes.Repository
		sandboxRepository          *fakes.SandboxRepository
		listenerFactory            *fakes.ListenerFactory
		dnsServerFactory           *fakes.DNSServerFactory
		command                    *fakes.Command
		ex                         executor.Executor
	)

	BeforeEach(func() {
		addressManager = &fakes.AddressManager{}
		routeManager = &fakes.RouteManager{}
		linkFactory = &fakes.LinkFactory{}
		sandboxNamespaceRepository = &fakes.Repository{}
		sandboxRepository = &fakes.SandboxRepository{}
		listenerFactory = &fakes.ListenerFactory{}
		dnsServerFactory = &fakes.DNSServerFactory{}

		command = &fakes.Command{}

		ex = executor.New(
			addressManager,
			routeManager,
			linkFactory,
			sandboxNamespaceRepository,
			sandboxRepository,
			listenerFactory,
			dnsServerFactory,
		)
	})

	It("executes a command with a context", func() {
		err := ex.Execute(command)
		Expect(err).NotTo(HaveOccurred())

		Expect(command.ExecuteCallCount()).To(Equal(1))
	})

	Describe("Context", func() {
		var context executor.Context

		BeforeEach(func() {
			c, ok := ex.(executor.Context)
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

		Describe("SandboxNamespaceRepository", func() {
			It("returns the SandboxNamespaceRepository", func() {
				Expect(context.SandboxNamespaceRepository()).To(Equal(sandboxNamespaceRepository))
			})
		})

		Describe("SandboxRepository", func() {
			It("returns the SandboxRepository", func() {
				Expect(context.SandboxRepository()).To(Equal(sandboxRepository))
			})
		})

		Describe("ListenerFactory", func() {
			It("returns the ListenerFactory", func() {
				Expect(context.ListenerFactory()).To(Equal(listenerFactory))
			})
		})

		Describe("DNSServerFactory", func() {
			It("returns the DNSServerFactory", func() {
				Expect(context.DNSServerFactory()).To(Equal(dnsServerFactory))
			})
		})
	})
})
