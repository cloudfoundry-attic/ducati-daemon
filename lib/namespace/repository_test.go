package namespace_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
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
		repo   namespace.Repository
		logger *lagertest.TestLogger
		dir    string
	)

	BeforeEach(func() {
		var err error
		dir, err = ioutil.TempDir("", "path")
		Expect(err).NotTo(HaveOccurred())

		logger = lagertest.NewTestLogger("test")

		repo, err = namespace.NewRepository(logger, dir)
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
		BeforeEach(func() {
			var err error
			repo, err = namespace.NewRepository(logger, "/var/run/netns")
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes the namespace bind mount and file and logs the operation", func() {
			err := exec.Command("ip", "netns", "add", "destroy-ns-test").Run()
			Expect(err).NotTo(HaveOccurred())

			ns, err := repo.Get("destroy-ns-test")
			Expect(err).NotTo(HaveOccurred())

			err = repo.Destroy(ns)
			Expect(err).NotTo(HaveOccurred())

			Expect("/var/run/netns/destroy-ns-test").NotTo(BeAnExistingFile())
			Expect(logger).To(gbytes.Say("destroy.destroying.*destroy-ns-test"))
		})

		Context("when the namespace is not located within this repository", func() {
			var ns namespace.Namespace

			BeforeEach(func() {
				tempFile, err := ioutil.TempFile(dir, "namespace")
				Expect(err).NotTo(HaveOccurred())
				Expect(tempFile.Close()).To(Succeed())

				ns = &namespace.Netns{File: tempFile}
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
			var nsPath string
			var nsFile *os.File

			BeforeEach(func() {
				Expect(os.MkdirAll("/var/run/netns", 0644)).To(Succeed())
				nsPath = filepath.Join("/var/run/netns", "simple-file")
				var err error
				nsFile, err = os.Create(nsPath)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				os.Remove(nsPath)
			})

			It("returns an error", func() {
				ns := &namespace.Netns{File: nsFile}
				err := repo.Destroy(ns)
				Expect(err).To(HaveOccurred())
			})

			It("does not remove the file", func() {
				Expect(nsPath).To(BeAnExistingFile())
			})
		})
	})
})
