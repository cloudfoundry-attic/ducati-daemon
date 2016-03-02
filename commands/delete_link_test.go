package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteLink", func() {
	var (
		context           *fakes.Context
		deleteLinkCommand commands.DeleteLink
		linkDeletor       *fakes.LinkDeletor
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		linkDeletor = &fakes.LinkDeletor{}

		context.LinkDeletorReturns(linkDeletor)

		deleteLinkCommand = commands.DeleteLink{LinkName: "some-link-name"}
	})

	It("calls Delete method on context.LinkDeletor", func() {
		err := deleteLinkCommand.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(linkDeletor.DeleteLinkByNameCallCount()).To(Equal(1))
		Expect(linkDeletor.DeleteLinkByNameArgsForCall(0)).To(Equal("some-link-name"))
	})

	It("returns the result of the LinkDeletor call", func() {
		linkDeletor.DeleteLinkByNameReturns(errors.New("whatever"))
		err := deleteLinkCommand.Execute(context)
		Expect(err).To(MatchError("whatever"))
	})
})