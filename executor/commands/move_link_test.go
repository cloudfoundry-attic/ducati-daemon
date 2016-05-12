package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MoveLink", func() {
	var (
		context           *fakes.Context
		linkFactory       *fakes.LinkFactory
		sandboxRepository *fakes.SandboxRepository
		sbox              *fakes.Sandbox
		moveLink          commands.MoveLink
	)

	BeforeEach(func() {
		context = &fakes.Context{}

		linkFactory = &fakes.LinkFactory{}
		context.LinkFactoryReturns(linkFactory)

		sandboxRepository = &fakes.SandboxRepository{}
		context.SandboxRepositoryReturns(sandboxRepository)

		sbox = &fakes.Sandbox{}
		sandboxRepository.GetReturns(sbox, nil)

		ns := &fakes.Namespace{}
		ns.FdReturns(999)
		ns.NameReturns("target-namespace")

		sbox.NamespaceReturns(ns)

		moveLink = commands.MoveLink{
			Name:        "link-name",
			SandboxName: "sandbox-name",
		}
	})

	It("gets the sandbox from the repository", func() {
		err := moveLink.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(sandboxRepository.GetCallCount()).To(Equal(1))
		Expect(sandboxRepository.GetArgsForCall(0)).To(Equal("sandbox-name"))
	})

	It("moves the link to the target namespace", func() {
		err := moveLink.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(linkFactory.SetNamespaceCallCount()).To(Equal(1))
		name, fd := linkFactory.SetNamespaceArgsForCall(0)
		Expect(name).To(Equal("link-name"))
		Expect(fd).To(BeEquivalentTo(999))
	})

	Context("when getting the sandbox fails", func() {
		BeforeEach(func() {
			sandboxRepository.GetReturns(nil, errors.New("welp"))
		})

		It("returns a meaningful error", func() {
			err := moveLink.Execute(context)
			Expect(err).To(MatchError("get sandbox: welp"))
		})
	})

	Context("when moving the link fails", func() {
		BeforeEach(func() {
			linkFactory.SetNamespaceReturns(errors.New("welp"))
		})

		It("returns a meaningful error", func() {
			err := moveLink.Execute(context)
			Expect(err).To(MatchError("move link: welp"))
		})
	})

	Describe("String", func() {
		It("describes itself", func() {
			Expect(moveLink.String()).To(Equal("ip link set dev link-name netns sandbox-name"))
		})
	})
})
