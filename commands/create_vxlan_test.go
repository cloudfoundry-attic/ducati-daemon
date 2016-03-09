package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateVxlan", func() {
	var (
		context      *fakes.Context
		vxlanFactory *fakes.VxlanFactory
		createVxlan  commands.CreateVxlan
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		vxlanFactory = &fakes.VxlanFactory{}
		context.VxlanFactoryReturns(vxlanFactory)

		createVxlan = commands.CreateVxlan{
			Name: "my-vxlan",
			VNI:  99,
		}
	})

	It("uses the factory to create the adapter", func() {
		err := createVxlan.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(vxlanFactory.CreateVxlanCallCount()).To(Equal(1))
		name, vni := vxlanFactory.CreateVxlanArgsForCall(0)
		Expect(name).To(Equal("my-vxlan"))
		Expect(vni).To(Equal(99))
	})

	Context("when the factory fails", func() {
		BeforeEach(func() {
			vxlanFactory.CreateVxlanReturns(errors.New("no vxlan for you"))
		})

		It("wraps and propagates the error", func() {
			err := createVxlan.Execute(context)
			Expect(err).To(MatchError("create vxlan: no vxlan for you"))
		})
	})

	Describe("String", func() {
		It("is self describing", func() {
			Expect(createVxlan.String()).To(Equal("ip link add my-vxlan type vxlan vni 99"))
		})
	})
})
