package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateBridge", func() {
	var (
		linkFactory  *fakes.LinkFactory
		context      *fakes.Context
		createBridge commands.CreateBridge
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		linkFactory = &fakes.LinkFactory{}
		context.LinkFactoryReturns(linkFactory)

		createBridge = commands.CreateBridge{
			Name: "my-bridge",
		}
	})

	It("creates a bridge device", func() {
		err := createBridge.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(linkFactory.CreateBridgeCallCount()).To(Equal(1))
		Expect(linkFactory.CreateBridgeArgsForCall(0)).To(Equal("my-bridge"))
	})

	Context("when creating the bridge fails", func() {
		BeforeEach(func() {
			linkFactory.CreateBridgeReturns(errors.New("no bridge for sale"))
		})

		It("wraps and propagates the error", func() {
			err := createBridge.Execute(context)
			Expect(err).To(MatchError("create bridge: no bridge for sale"))
		})
	})

	Describe("String", func() {
		It("is self describing", func() {
			Expect(createBridge.String()).To(Equal("ip link add dev my-bridge type bridge"))
		})
	})
})
