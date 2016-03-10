package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateSandboxNamespace", func() {
	var (
		context                *fakes.Context
		repository             *fakes.Repository
		createSandboxNamespace commands.CreateSandboxNamespace
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		repository = &fakes.Repository{}
		context.SandboxRepositoryReturns(repository)

		createSandboxNamespace = commands.CreateSandboxNamespace{
			Name: "my-namespace",
		}

		repository.CreateReturns(namespace.NewNamespace("/some/path"), nil)
	})

	It("creates the namespace in the repository", func() {
		err := createSandboxNamespace.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(repository.CreateCallCount()).To(Equal(1))
		Expect(repository.CreateArgsForCall(0)).To(Equal("my-namespace"))
	})

	Context("when creating the namespace fails", func() {
		BeforeEach(func() {
			repository.CreateReturns(nil, errors.New("welp"))
		})

		It("wraps and propogates the error", func() {
			err := createSandboxNamespace.Execute(context)
			Expect(err).To(MatchError("create namespace: welp"))
		})
	})

	Describe("String", func() {
		It("is self describing", func() {
			Expect(createSandboxNamespace.String()).To(Equal("ip netns add my-namespace"))
		})
	})
})
