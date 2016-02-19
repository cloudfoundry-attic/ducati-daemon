package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateVeth", func() {
	var (
		context     *fakes.Context
		vethFactory *fakes.VethFactory
		createVeth  commands.CreateVeth
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		vethFactory = &fakes.VethFactory{}
		context.VethFactoryReturns(vethFactory)

		createVeth = commands.CreateVeth{
			Name:     "if-name",
			PeerName: "peer-if-name",
			MTU:      99,
		}
	})

	It("uses the link factory to create the veth pair", func() {
		err := createVeth.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		name, peerName, mtu := vethFactory.CreateVethArgsForCall(0)
		Expect(name).To(Equal("if-name"))
		Expect(peerName).To(Equal("peer-if-name"))
		Expect(mtu).To(Equal(99))
	})

	Context("when creating the link fails", func() {
		It("propogates the error", func() {
			vethFactory.CreateVethReturns(errors.New("welp"))

			err := createVeth.Execute(context)
			Expect(err).To(MatchError("failed to create veth pair if-name:peer-if-name: welp"))
		})
	})
})
