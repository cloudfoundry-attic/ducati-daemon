package conditions_test

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/conditions"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/sandbox"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SandboxExists", func() {
	var (
		repo          *fakes.SandboxRepository
		sandboxExists conditions.SandboxExists
		context       *fakes.Context
	)

	BeforeEach(func() {
		repo = &fakes.SandboxRepository{}
		context = &fakes.Context{}
		context.SandboxRepositoryReturns(repo)

		sandboxExists = conditions.SandboxExists{
			Name: "some-sandbox",
		}
	})

	Context("when the namespace exists", func() {
		BeforeEach(func() {
			repo.GetReturns(&sandbox.Sandbox{})
		})

		It("returns true", func() {
			Expect(sandboxExists.Satisfied(context)).To(BeTrue())

			Expect(repo.GetCallCount()).To(Equal(1))
			Expect(repo.GetArgsForCall(0)).To(Equal("some-sandbox"))
		})
	})

	Context("when the namespace does not exist", func() {
		BeforeEach(func() {
			repo.GetReturns(nil)
		})

		It("returns false", func() {
			Expect(sandboxExists.Satisfied(context)).To(BeFalse())

			Expect(repo.GetCallCount()).To(Equal(1))
			Expect(repo.GetArgsForCall(0)).To(Equal("some-sandbox"))
		})
	})

	Context("String", func() {
		It("describes itself", func() {
			Expect(sandboxExists.String()).To(Equal(`check if sandbox "some-sandbox" exists`))
		})
	})
})
