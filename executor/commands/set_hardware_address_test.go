package commands_test

import (
	"errors"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SetHardwareAddress", func() {
	var (
		context     *fakes.Context
		linkFactory *fakes.LinkFactory
		hwAddr      net.HardwareAddr

		setHWAddress *commands.SetHardwareAddress
	)

	BeforeEach(func() {
		var err error
		hwAddr, err = net.ParseMAC("01:02:03:04:05:06")
		Expect(err).NotTo(HaveOccurred())

		linkFactory = &fakes.LinkFactory{}
		linkFactory.HardwareAddressReturns(hwAddr, nil)

		context = &fakes.Context{}
		context.LinkFactoryReturns(linkFactory)

		setHWAddress = &commands.SetHardwareAddress{
			LinkName:        "some-link-name",
			HardwareAddress: hwAddr,
		}
	})

	It("sets the hardware address", func() {
		err := setHWAddress.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(linkFactory.SetHardwareAddressCallCount()).To(Equal(1))
		linkName, addr := linkFactory.SetHardwareAddressArgsForCall(0)
		Expect(linkName).To(Equal("some-link-name"))
		Expect(addr).To(Equal(hwAddr))
	})

	Context("when setting the hardware address fails", func() {
		BeforeEach(func() {
			linkFactory.SetHardwareAddressReturns(errors.New("boom"))
		})

		It("wraps and propogates the error", func() {
			err := setHWAddress.Execute(context)
			Expect(err).To(MatchError("set hardware address: boom"))
		})
	})

	Describe("String", func() {
		It("describes itself", func() {
			Expect(setHWAddress.String()).To(Equal("ip link set some-link-name address 01:02:03:04:05:06"))
		})
	})
})
