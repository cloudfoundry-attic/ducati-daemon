package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateNamespace", func() {
	var (
		context         *fakes.Context
		repository      *fakes.Repository
		createNamespace *commands.CreateNamespace
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		repository = &fakes.Repository{}

		createNamespace = &commands.CreateNamespace{
			Name:       "my-namespace",
			Repository: repository,
		}

		repository.CreateReturns(namespace.NewNamespace("/some/path"), nil)
	})

	It("creates the namespace in the repository", func() {
		err := createNamespace.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(createNamespace.Result).NotTo(BeNil())

		Expect(repository.CreateCallCount()).To(Equal(1))
		Expect(repository.CreateArgsForCall(0)).To(Equal("my-namespace"))
	})

	Context("when creating the namespace fails", func() {
		BeforeEach(func() {
			repository.CreateReturns(nil, errors.New("welp"))
		})

		It("returns a meaningful error", func() {
			err := createNamespace.Execute(context)
			Expect(err).To(MatchError(`failed to create namespace "my-namespace": welp`))
		})
	})
})
