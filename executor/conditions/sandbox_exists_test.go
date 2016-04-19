package conditions_test

import (
	"errors"

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
			repo.GetReturns(&fakes.Sandbox{}, nil)
		})

		It("returns true", func() {
			satisfied, err := sandboxExists.Satisfied(context)
			Expect(err).NotTo(HaveOccurred())
			Expect(satisfied).To(BeTrue())

			Expect(repo.GetCallCount()).To(Equal(1))
			Expect(repo.GetArgsForCall(0)).To(Equal("some-sandbox"))
		})
	})

	Context("when the namespace does not exist", func() {
		BeforeEach(func() {
			repo.GetReturns(nil, sandbox.NotFoundError)
		})

		It("returns false", func() {
			satisfied, err := sandboxExists.Satisfied(context)
			Expect(err).NotTo(HaveOccurred())
			Expect(satisfied).To(BeFalse())

			Expect(repo.GetCallCount()).To(Equal(1))
			Expect(repo.GetArgsForCall(0)).To(Equal("some-sandbox"))
		})
	})

	Context("when an unknown error occurs", func() {
		BeforeEach(func() {
			repo.GetReturns(nil, errors.New("some error"))
		})
		It("wraps and returns the error", func() {
			_, err := sandboxExists.Satisfied(context)
			Expect(err).To(MatchError("sandbox get: some error"))
		})

	})

	Context("String", func() {
		It("describes itself", func() {
			Expect(sandboxExists.String()).To(Equal(`check if sandbox "some-sandbox" exists`))
		})
	})
})
