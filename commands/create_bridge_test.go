package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateBridge", func() {
	var (
		bridgeFactory *fakes.BridgeFactory
		context       *fakes.Context
		createBridge  commands.CreateBridge
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		bridgeFactory = &fakes.BridgeFactory{}
		context.BridgeFactoryReturns(bridgeFactory)

		createBridge = commands.CreateBridge{
			Name: "my-bridge",
		}
	})

	It("creates a bridge device", func() {
		err := createBridge.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(bridgeFactory.CreateBridgeCallCount()).To(Equal(1))
		Expect(bridgeFactory.CreateBridgeArgsForCall(0)).To(Equal("my-bridge"))
	})

	Context("when the bridge factory fails", func() {
		BeforeEach(func() {
			bridgeFactory.CreateBridgeReturns(errors.New("no bridge for sale"))
		})

		It("propagates the error", func() {
			err := createBridge.Execute(context)
			Expect(err).To(MatchError("no bridge for sale"))
		})
	})
})
