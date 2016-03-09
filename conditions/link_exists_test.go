package conditions_test

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/conditions"
	"github.com/cloudfoundry-incubator/ducati-daemon/conditions/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LinkExists", func() {
	var (
		linkFinder *fakes.LinkFinder
		linkExists conditions.LinkExists
	)

	BeforeEach(func() {
		linkFinder = &fakes.LinkFinder{}

		linkExists = conditions.LinkExists{
			LinkFinder: linkFinder,
			Name:       "my-interface",
		}
	})

	Context("when the link exists", func() {
		BeforeEach(func() {
			linkFinder.ExistsReturns(true)
		})

		It("returns true", func() {
			Expect(linkExists.Satisfied(nil)).To(BeTrue())
			Expect(linkFinder.ExistsCallCount()).To(Equal(1))
			Expect(linkFinder.ExistsArgsForCall(0)).To(Equal("my-interface"))
		})
	})

	Context("when the link does not exist", func() {
		BeforeEach(func() {
			linkFinder.ExistsReturns(false)
		})

		It("returns false", func() {
			Expect(linkExists.Satisfied(nil)).To(BeFalse())
			Expect(linkFinder.ExistsCallCount()).To(Equal(1))
			Expect(linkFinder.ExistsArgsForCall(0)).To(Equal("my-interface"))
		})
	})

	Context("String", func() {
		It("describes itself", func() {
			Expect(linkExists.String()).To(Equal(`check if link "my-interface" exists`))
		})
	})
})
