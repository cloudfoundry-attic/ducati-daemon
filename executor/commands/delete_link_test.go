package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteLink", func() {
	var (
		context           *fakes.Context
		deleteLinkCommand commands.DeleteLink
		linkFactory       *fakes.LinkFactory
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		linkFactory = &fakes.LinkFactory{}

		context.LinkFactoryReturns(linkFactory)

		deleteLinkCommand = commands.DeleteLink{LinkName: "some-link-name"}
	})

	It("calls Delete method on context.LinkFactory", func() {
		err := deleteLinkCommand.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(linkFactory.DeleteLinkByNameCallCount()).To(Equal(1))
		Expect(linkFactory.DeleteLinkByNameArgsForCall(0)).To(Equal("some-link-name"))
	})

	Context("when deleting the link by name fails", func() {
		BeforeEach(func() {
			linkFactory.DeleteLinkByNameReturns(errors.New("whatever"))
		})

		It("wraps and propogates the error", func() {
			err := deleteLinkCommand.Execute(context)
			Expect(err).To(MatchError("delete link: whatever"))
		})
	})

	Describe("String", func() {
		It("describes itself", func() {
			Expect(deleteLinkCommand.String()).To(Equal("ip link del some-link-name"))
		})
	})
})
