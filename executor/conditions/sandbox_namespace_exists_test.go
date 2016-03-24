package conditions_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/conditions"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SandboxNamespaceExists", func() {
	var (
		repo            *fakes.Repository
		namespaceExists conditions.SandboxNamespaceExists
		context         *fakes.Context
	)

	BeforeEach(func() {
		repo = &fakes.Repository{}
		context = &fakes.Context{}
		context.SandboxRepositoryReturns(repo)

		namespaceExists = conditions.SandboxNamespaceExists{
			Name: "some-sandbox",
		}
	})

	Context("when the namespace exists", func() {
		BeforeEach(func() {
			repo.GetReturns(&fakes.Namespace{}, nil)
		})

		It("returns true", func() {
			Expect(namespaceExists.Satisfied(context)).To(BeTrue())

			Expect(repo.GetCallCount()).To(Equal(1))
			Expect(repo.GetArgsForCall(0)).To(Equal("some-sandbox"))
		})
	})

	Context("when the namespace does not exist", func() {
		BeforeEach(func() {
			repo.GetReturns(nil, errors.New("nope"))
		})

		It("returns false", func() {
			Expect(namespaceExists.Satisfied(context)).To(BeFalse())

			Expect(repo.GetCallCount()).To(Equal(1))
			Expect(repo.GetArgsForCall(0)).To(Equal("some-sandbox"))
		})
	})

	Context("String", func() {
		It("describes itself", func() {
			Expect(namespaceExists.String()).To(Equal(`check if sandbox "some-sandbox" exists`))
		})
	})
})
