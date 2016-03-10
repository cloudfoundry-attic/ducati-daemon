package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateVeth", func() {
	var (
		context     *fakes.Context
		linkFactory *fakes.LinkFactory
		createVeth  commands.CreateVeth
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		linkFactory = &fakes.LinkFactory{}
		context.LinkFactoryReturns(linkFactory)

		createVeth = commands.CreateVeth{
			Name:     "if-name",
			PeerName: "peer-if-name",
			MTU:      99,
		}
	})

	It("uses the link factory to create the veth pair", func() {
		err := createVeth.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		name, peerName, mtu := linkFactory.CreateVethArgsForCall(0)
		Expect(name).To(Equal("if-name"))
		Expect(peerName).To(Equal("peer-if-name"))
		Expect(mtu).To(Equal(99))
	})

	Context("when creating the link fails", func() {
		BeforeEach(func() {
			linkFactory.CreateVethReturns(errors.New("welp"))
		})

		It("wraps and propogates the error", func() {
			err := createVeth.Execute(context)
			Expect(err).To(MatchError("create veth: welp"))
		})
	})

	Describe("String", func() {
		It("is self describing", func() {
			Expect(createVeth.String()).To(Equal("ip link add dev if-name mtu 99 type veth peer name peer-if-name mtu 99"))
		})
	})
})
