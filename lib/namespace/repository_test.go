package namespace_test

import (
	"io/ioutil"
	"path/filepath"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Repository", func() {
	Describe("PathOf", func() {
		It("returns the full path of a namespace in the repository", func() {
			dir, err := ioutil.TempDir("", "path")

			Expect(err).NotTo(HaveOccurred())

			repo, err := namespace.NewRepository(dir)
			Expect(err).NotTo(HaveOccurred())

			Expect(repo.PathOf("some-namespace")).To(Equal(filepath.Join(dir, "some-namespace")))
		})
	})
})
