package executor_test

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("DNS Factory", func() {
	Describe("DNS New", func() {
		var (
			decoratorFactory *fakes.WriterDecoratorFactory
			logger           *lagertest.TestLogger
			ns               *fakes.Namespace
			dnsFactory       *executor.DNSFactory
		)

		BeforeEach(func() {
			decoratorFactory = &fakes.WriterDecoratorFactory{}
			ns = &fakes.Namespace{}
			logger = lagertest.NewTestLogger("test")

			dnsFactory = &executor.DNSFactory{
				Logger:           logger,
				DecoratorFactory: decoratorFactory,
			}
		})

		It("calls the DecoratorFactory with sandboxNS", func() {
			dnsFactory.New(nil, ns)

			Expect(decoratorFactory.DecorateCallCount()).To(Equal(1))
			Expect(decoratorFactory.DecorateArgsForCall(0)).To(Equal(ns))
		})
	})
})
