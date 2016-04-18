package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/sandbox"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateSandbox", func() {
	var (
		context           *fakes.Context
		sandboxRepository *fakes.SandboxRepository
		createSandbox     commands.CreateSandbox
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		sandboxRepository = &fakes.SandboxRepository{}
		context.SandboxRepositoryReturns(sandboxRepository)

		createSandbox = commands.CreateSandbox{
			Name: "my-namespace",
		}

		sandbox := &sandbox.Sandbox{}
		sandboxRepository.CreateReturns(sandbox, nil)
	})

	It("creates the sandbox in the repository", func() {
		err := createSandbox.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(sandboxRepository.CreateCallCount()).To(Equal(1))
		Expect(sandboxRepository.CreateArgsForCall(0)).To(Equal("my-namespace"))
	})

	Context("when creating the namespace fails", func() {
		BeforeEach(func() {
			sandboxRepository.CreateReturns(nil, errors.New("welp"))
		})

		It("wraps and propogates the error", func() {
			err := createSandbox.Execute(context)
			Expect(err).To(MatchError("create sandbox: welp"))
		})
	})

	Describe("String", func() {
		It("is self describing", func() {
			Expect(createSandbox.String()).To(Equal("create sandbox my-namespace"))
		})
	})
})
