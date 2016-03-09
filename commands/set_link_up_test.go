package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SetLinkUp", func() {
	var (
		context   *fakes.Context
		setUpper  *fakes.SetUpper
		setLinkUp commands.SetLinkUp
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		setUpper = &fakes.SetUpper{}
		context.SetUpperReturns(setUpper)

		setLinkUp = commands.SetLinkUp{
			LinkName: "link-name",
		}
	})

	It("sets the link up", func() {
		err := setLinkUp.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(setUpper.SetUpCallCount()).To(Equal(1))
		Expect(setUpper.SetUpArgsForCall(0)).To(Equal("link-name"))
	})

	Context("when setting the link UP fails", func() {
		It("wraps and propagates the error", func() {
			setUpper.SetUpReturns(errors.New("welp"))

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
