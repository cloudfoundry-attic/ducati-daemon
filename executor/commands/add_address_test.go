package commands_test

import (
	"errors"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddAddress", func() {
	var (
		addressManager *fakes.AddressManager
		context        *fakes.Context
		addAddress     commands.AddAddress
	)

	BeforeEach(func() {
		addressManager = &fakes.AddressManager{}
		context = &fakes.Context{}
		context.AddressManagerReturns(addressManager)

		addAddress = commands.AddAddress{
			InterfaceName: "my-interface",
			Address: net.IPNet{
				IP:   net.ParseIP("192.168.1.1"),
				Mask: net.CIDRMask(24, 32),
			},
		}
	})

	It("adds the address to the interface", func() {
		err := addAddress.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(addressManager.AddAddressCallCount()).To(Equal(1))
		interfaceName, address := addressManager.AddAddressArgsForCall(0)
		Expect(interfaceName).To(Equal("my-interface"))
		Expect(address.String()).To(Equal("192.168.1.1/24"))
	})

	Context("when adding the address fails", func() {
		BeforeEach(func() {
			addressManager.AddAddressReturns(errors.New("no address for you"))
		})

		It("wraps and propagates the error", func() {
			err := addAddress.Execute(context)
			Expect(err).To(MatchError("add address: no address for you"))
		})
	})

	Describe("String", func() {
		It("describes itself", func() {
			Expect(addAddress.String()).To(Equal("ip addr add 192.168.1.1/24 dev my-interface"))
		})
	})
})
