package conditions_test

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/conditions"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LinkExists", func() {
	var (
		context     *fakes.Context
		linkFactory *fakes.LinkFactory
		linkExists  conditions.LinkExists
	)

	BeforeEach(func() {
		context = &fakes.Context{}

		linkFactory = &fakes.LinkFactory{}
		context.LinkFactoryReturns(linkFactory)

		linkExists = conditions.LinkExists{
			Name: "my-interface",
		}
	})

	Context("when the link exists", func() {
		BeforeEach(func() {
			linkFactory.ExistsReturns(true)
		})

		It("returns true", func() {
			Expect(linkExists.Satisfied(context)).To(BeTrue())
			Expect(linkFactory.ExistsCallCount()).To(Equal(1))
			Expect(linkFactory.ExistsArgsForCall(0)).To(Equal("my-interface"))
		})
	})

	Context("when the link does not exist", func() {
		BeforeEach(func() {
			linkFactory.ExistsReturns(false)
		})

		It("returns false", func() {
			Expect(linkExists.Satisfied(context)).To(BeFalse())
			Expect(linkFactory.ExistsCallCount()).To(Equal(1))
			Expect(linkFactory.ExistsArgsForCall(0)).To(Equal("my-interface"))
		})
	})

	Context("String", func() {
		It("describes itself", func() {
			Expect(linkExists.String()).To(Equal(`check if link "my-interface" exists`))
		})
	})
})
