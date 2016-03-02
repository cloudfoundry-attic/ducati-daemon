package conditions_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/conditions"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NamespaceExists", func() {
	var (
		repo            *fakes.Repository
		namespaceExists conditions.NamespaceExists
	)

	BeforeEach(func() {
		repo = &fakes.Repository{}
		namespaceExists = conditions.NamespaceExists{
			Name:       "namespace",
			Repository: repo,
		}
	})

	Context("when the namespace exists", func() {
		BeforeEach(func() {
			repo.GetReturns(namespace.NewNamespace("namespace"), nil)
		})

		It("returns true", func() {
			Expect(namespaceExists.Satisfied(nil)).To(BeTrue())

			Expect(repo.GetCallCount()).To(Equal(1))
			Expect(repo.GetArgsForCall(0)).To(Equal("namespace"))
		})
	})

	Context("when the namespace does not exist", func() {
		BeforeEach(func() {
			repo.GetReturns(nil, errors.New("nope"))
		})

		It("returns false", func() {
			Expect(namespaceExists.Satisfied(nil)).To(BeFalse())

			Expect(repo.GetCallCount()).To(Equal(1))
			Expect(repo.GetArgsForCall(0)).To(Equal("namespace"))
		})
	})
})
