package commands_test

import (
	"errors"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetHardwareAddress", func() {
	var (
		context           *fakes.Context
		hardwareAddresser *fakes.HardwareAddresser
		hwAddr            net.HardwareAddr

		getHWAddress *commands.GetHardwareAddress
	)

	BeforeEach(func() {
		var err error
		hwAddr, err = net.ParseMAC("FF:FF:FF:FF:FF:FF")
		Expect(err).NotTo(HaveOccurred())

		hardwareAddresser = &fakes.HardwareAddresser{}
		hardwareAddresser.HardwareAddressReturns(hwAddr, nil)

		context = &fakes.Context{}
		context.HardwareAddresserReturns(hardwareAddresser)

		getHWAddress = &commands.GetHardwareAddress{
			LinkName: "some-link-name",
		}
	})

	It("gets the hardware address", func() {
		err := getHWAddress.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(hardwareAddresser.HardwareAddressCallCount()).To(Equal(1))
		Expect(hardwareAddresser.HardwareAddressArgsForCall(0)).To(Equal("some-link-name"))

		Expect(getHWAddress.Result).To(Equal(hwAddr))
	})

	Context("when getting the hardware address fails", func() {
		BeforeEach(func() {
			hardwareAddresser.HardwareAddressReturns(nil, errors.New("boom"))
		})

		It("returns the error", func() {
			err := getHWAddress.Execute(context)
			Expect(err).To(MatchError("boom"))
		})
	})
})
