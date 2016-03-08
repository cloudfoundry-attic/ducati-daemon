package commands_test

import (
	"errors"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddAddress", func() {
	var (
		addressAdder *fakes.AddressAdder
		context      *fakes.Context
		addAddress   commands.AddAddress
	)

	BeforeEach(func() {
		addressAdder = &fakes.AddressAdder{}
		context = &fakes.Context{}
		context.AddressAdderReturns(addressAdder)

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

		Expect(addressAdder.AddAddressCallCount()).To(Equal(1))
		interfaceName, address := addressAdder.AddAddressArgsForCall(0)
		Expect(interfaceName).To(Equal("my-interface"))
		Expect(address.String()).To(Equal("192.168.1.1/24"))
	})

	Context("when the address adder fails", func() {
		BeforeEach(func() {
			addressAdder.AddAddressReturns(errors.New("no address for you"))
		})

		It("wraps and propagates the error", func() {
			err := addAddress.Execute(context)
			Expect(err).To(MatchError("add address: no address for you"))
		})
	})
})
