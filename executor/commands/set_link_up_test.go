package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SetLinkUp", func() {
	var (
		context     *fakes.Context
		linkFactory *fakes.LinkFactory
		setLinkUp   commands.SetLinkUp
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		linkFactory = &fakes.LinkFactory{}
		context.LinkFactoryReturns(linkFactory)

		setLinkUp = commands.SetLinkUp{
			LinkName: "link-name",
		}
	})

	It("sets the link up", func() {
		err := setLinkUp.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(linkFactory.SetUpCallCount()).To(Equal(1))
		Expect(linkFactory.SetUpArgsForCall(0)).To(Equal("link-name"))
	})

	Context("when setting the link UP fails", func() {
		It("wraps and propagates the error", func() {
			linkFactory.SetUpReturns(errors.New("welp"))

			err := setLinkUp.Execute(context)
			Expect(err).To(MatchError("set link up: welp"))
		})
	})

	Describe("String", func() {
		It("describes itself", func() {
			Expect(setLinkUp.String()).To(Equal("ip link set link-name up"))
		})
	})
})
