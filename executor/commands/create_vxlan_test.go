package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateVxlan", func() {
	var (
		context     *fakes.Context
		linkFactory *fakes.LinkFactory
		createVxlan commands.CreateVxlan
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		linkFactory = &fakes.LinkFactory{}
		context.LinkFactoryReturns(linkFactory)

		createVxlan = commands.CreateVxlan{
			Name: "my-vxlan",
			VNI:  99,
		}
	})

	It("uses the factory to create the adapter", func() {
		err := createVxlan.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(linkFactory.CreateVxlanCallCount()).To(Equal(1))
		name, vni := linkFactory.CreateVxlanArgsForCall(0)
		Expect(name).To(Equal("my-vxlan"))
		Expect(vni).To(Equal(99))
	})

	Context("when creating the vxlan link fails", func() {
		BeforeEach(func() {
			linkFactory.CreateVxlanReturns(errors.New("no vxlan for you"))
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
