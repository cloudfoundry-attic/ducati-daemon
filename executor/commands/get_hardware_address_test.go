package commands_test

import (
	"errors"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetHardwareAddress", func() {
	var (
		context     *fakes.Context
		linkFactory *fakes.LinkFactory
		hwAddr      net.HardwareAddr

		getHWAddress *commands.GetHardwareAddress
	)

	BeforeEach(func() {
		var err error
		hwAddr, err = net.ParseMAC("FF:FF:FF:FF:FF:FF")
		Expect(err).NotTo(HaveOccurred())

		linkFactory = &fakes.LinkFactory{}
		linkFactory.HardwareAddressReturns(hwAddr, nil)

		context = &fakes.Context{}
		context.LinkFactoryReturns(linkFactory)

		getHWAddress = &commands.GetHardwareAddress{
			LinkName: "some-link-name",
		}
	})

	It("gets the hardware address", func() {
		err := getHWAddress.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(linkFactory.HardwareAddressCallCount()).To(Equal(1))
		Expect(linkFactory.HardwareAddressArgsForCall(0)).To(Equal("some-link-name"))

		Expect(getHWAddress.Result).To(Equal(hwAddr))
	})

	Context("when getting the hardware address fails", func() {
		BeforeEach(func() {
			linkFactory.HardwareAddressReturns(nil, errors.New("boom"))
		})

		It("wraps and propogates the error", func() {
			err := getHWAddress.Execute(context)
			Expect(err).To(MatchError("get hardware address: boom"))
		})
	})

	Describe("String", func() {
		It("describes itself", func() {
			Expect(getHWAddress.String()).To(Equal("ip link show some-link-name"))
		})
	})
})
