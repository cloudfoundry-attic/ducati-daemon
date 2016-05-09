package namespace_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Repository", func() {
	var (
		repo         namespace.Repository
		logger       *lagertest.TestLogger
		threadLocker *fakes.OSThreadLocker
		dir          string
	)

	BeforeEach(func() {
		var err error
		dir, err = ioutil.TempDir("", "path")
		Expect(err).NotTo(HaveOccurred())

		logger = lagertest.NewTestLogger("test")
		threadLocker = &fakes.OSThreadLocker{}
		repo, err = namespace.NewRepository(logger, dir, threadLocker)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(dir)
	})

	Describe("PathOf", func() {
		It("returns the full path of a namespace in the repository", func() {
			Expect(repo.PathOf("some-namespace")).To(Equal(filepath.Join(dir, "some-namespace")))
		})
	})

	Describe("Destroy", func() {
		It("removes the namespace bind mount and file and logs the operation", func() {
			ns, err := repo.Create("destroy-ns-test")
			Expect(err).NotTo(HaveOccurred())

			err = repo.Destroy(ns)
			Expect(err).NotTo(HaveOccurred())

			Expect(path.Join(dir, "destroy-ns-test")).NotTo(BeAnExistingFile())
			Expect(logger).To(gbytes.Say("destroy.destroying.*destroy-ns-test"))
		})

		Context("when the namespace is not located within this repository", func() {
			var (
				ns          namespace.Namespace
				anotherRepo namespace.Repository
				repoDir     string
			)

			BeforeEach(func() {
				var err error
				repoDir, err = ioutil.TempDir("", "repo")
				Expect(err).NotTo(HaveOccurred())

				anotherRepo, err = namespace.NewRepository(logger, repoDir, threadLocker)
				Expect(err).NotTo(HaveOccurred())

				ns, err = anotherRepo.Create("outside")
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err := anotherRepo.Destroy(ns)
				Expect(err).NotTo(HaveOccurred())

				err = os.RemoveAll(repoDir)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns a meaningful error", func() {
				err := repo.Destroy(ns)
				Expect(err).To(MatchError(HavePrefix("namespace outside of repository:")))
			})

			It("logs the failure", func() {
				repo.Destroy(ns)
				Expect(logger).To(gbytes.Say("destroy.outside-of-repo.*name.*"))
			})
		})

		Context("when the namespace file does not exist", func() {
			var ns namespace.Namespace

			BeforeEach(func() {
				var err error
				ns, err = repo.Create(fmt.Sprintf("some-nonexistent-namespace-%d", GinkgoParallelNode()))
				Expect(err).NotTo(HaveOccurred())

				err = repo.Destroy(ns)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error", func() {
				err := repo.Destroy(ns)
				Expect(err).To(HaveOccurred())
			})

			It("logs the failure", func() {
				repo.Destroy(ns)
				Expect(logger).To(gbytes.Say("destroy.unlink-failed"))
			})
		})

		Context("when the namespace isn't a Netns", func() {
			var ns namespace.Namespace

			BeforeEach(func() {
				ns = &fakes.Namespace{}
			})

			It("returns an error", func() {
				err := repo.Destroy(ns)
				Expect(err).To(MatchError("namespace is not a Netns"))
			})

			It("logs the failure", func() {
				repo.Destroy(ns)
				Expect(logger).To(gbytes.Say("destroy.not-a-netns"))
			})
		})

		Context("when the namespace file is not a bind mount", func() {
			var ns namespace.Namespace

			BeforeEach(func() {
				var err error
				ns, err = repo.Create("already-destroyed")
				Expect(err).NotTo(HaveOccurred())

				err = repo.Destroy(ns)
				Expect(err).NotTo(HaveOccurred())

				f, err := os.Create(ns.Name())
				Expect(err).NotTo(HaveOccurred())
				f.Close()
			})

			AfterEach(func() {
				os.Remove(ns.Name())
			})

			It("returns an error", func() {
				err := repo.Destroy(ns)
				Expect(err).To(HaveOccurred())
			})

			It("does not remove the file", func() {
				Expect(ns.Name()).To(BeAnExistingFile())
			})
		})
	})
})
