package namespace_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Repository", func() {
	var (
		repo namespace.Repository
		dir  string
	)

	BeforeEach(func() {
		var err error
		dir, err = ioutil.TempDir("", "path")
		Expect(err).NotTo(HaveOccurred())

		repo, err = namespace.NewRepository(dir)
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
			repo, err = namespace.NewRepository("/var/run/netns")
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes the namespace bind mount and file", func() {
			err := exec.Command("ip", "netns", "add", "destroy-ns-test").Run()
			Expect(err).NotTo(HaveOccurred())

			ns, err := repo.Get("destroy-ns-test")
			Expect(err).NotTo(HaveOccurred())

			err = repo.Destroy(ns)
			Expect(err).NotTo(HaveOccurred())

			Expect("/var/run/netns/destroy-ns-test").NotTo(BeAnExistingFile())
		})

		Context("when the namespace is not located within this repository", func() {
			It("returns a meaningful error", func() {
				tempFile, err := ioutil.TempFile(dir, "namespace")
				Expect(err).NotTo(HaveOccurred())
				Expect(tempFile.Close()).To(Succeed())

				ns := &namespace.Netns{File: tempFile}

				err = repo.Destroy(ns)
				Expect(err).To(MatchError(HavePrefix("namespace outside of repository:")))
			})
		})

		Context("when the namespace file does not exist", func() {
			It("returns an error", func() {
				tempFile, err := ioutil.TempFile(dir, "namespace")
				Expect(err).NotTo(HaveOccurred())
				Expect(tempFile.Close()).To(Succeed())
				Expect(os.Remove(tempFile.Name())).To(Succeed())

				ns := &namespace.Netns{File: tempFile}

				err = repo.Destroy(ns)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the namespace isn't a Netns", func() {
			It("returns an error", func() {
				ns := &fakes.Namespace{}

				err := repo.Destroy(ns)
				Expect(err).To(MatchError("namespace is not a Netns"))
			})
		})

		Context("when the namespace file is not a bind mount", func() {
			var (
				nsPath string
				nsFile *os.File
			)

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
