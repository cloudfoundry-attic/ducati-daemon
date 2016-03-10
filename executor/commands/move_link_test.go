package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MoveLink", func() {
	var (
		context          *fakes.Context
		setNamespacer    *fakes.SetNamespacer
		setLinkNamespace commands.MoveLink
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		setNamespacer = &fakes.SetNamespacer{}
		context.SetNamespacerReturns(setNamespacer)

		setLinkNamespace = commands.MoveLink{
			Name:      "link-name",
			Namespace: "some-namespace-path",
		}
	})

	It("moves the link to the target namespace", func() {
		err := setLinkNamespace.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(setNamespacer.SetNamespaceCallCount()).To(Equal(1))
		name, ns := setNamespacer.SetNamespaceArgsForCall(0)
		Expect(name).To(Equal("link-name"))
		Expect(ns).To(Equal("some-namespace-path"))
	})

	Context("when moving the link fails", func() {
		It("wraps and propagates the error", func() {
			setNamespacer.SetNamespaceReturns(errors.New("welp"))

			err := setLinkNamespace.Execute(context)
			Expect(err).To(MatchError("move link: welp"))
		})
	})

	Describe("String", func() {
		It("describes itself", func() {
			Expect(setLinkNamespace.String()).To(Equal("ip link set dev link-name netns some-namespace-path"))
		})
	})
})
