package namespace_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("NamespaceRepo", func() {
	var repoDir string
	var logger *lagertest.TestLogger
	var threadLocker *fakes.OSThreadLocker

	BeforeEach(func() {
		var err error
		repoDir, err = ioutil.TempDir("", "ns-repo")
		Expect(err).NotTo(HaveOccurred())

		logger = lagertest.NewTestLogger("test")
		threadLocker = &fakes.OSThreadLocker{}
	})

	AfterEach(func() {
		err := os.RemoveAll(repoDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("NewRepository", func() {
		It("returns a repository", func() {
			repo, err := namespace.NewRepository(logger, repoDir, threadLocker)
			Expect(err).NotTo(HaveOccurred())
			Expect(repo).NotTo(BeNil())
		})

		Context("when the target directory does not exist", func() {
			BeforeEach(func() {
				err := os.RemoveAll(repoDir)
				Expect(err).NotTo(HaveOccurred())
			})

			It("creates the directory", func() {
				_, err := namespace.NewRepository(logger, repoDir, threadLocker)
				Expect(err).NotTo(HaveOccurred())

				info, err := os.Stat(repoDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(info.IsDir()).To(BeTrue())
			})
		})
	})

	Describe("Create", func() {
		var repo namespace.Repository
		var name string

		BeforeEach(func() {
			var err error
			repo, err = namespace.NewRepository(logger, repoDir, threadLocker)
			Expect(err).NotTo(HaveOccurred())

			name = fmt.Sprintf("test-ns-%d", GinkgoParallelNode())
		})

		It("creates a namespace in the repository", func() {
			ns, err := repo.Create(name)
			Expect(err).NotTo(HaveOccurred())

			nsPath := filepath.Join(repoDir, name)
			defer unix.Unmount(nsPath, unix.MNT_DETACH)

			Expect(ns.Name()).To(Equal(nsPath))

			var repoStat unix.Stat_t
			err = unix.Stat(nsPath, &repoStat)
			Expect(err).NotTo(HaveOccurred())

			var namespaceInode string
			callback := func(_ *os.File) error {
				output, err := exec.Command("stat", "-L", "-c", "%i", "/proc/self/ns/net").CombinedOutput()
				namespaceInode = strings.TrimSpace(string(output))
				return err
			}

			err = ns.Execute(callback)
			Expect(err).NotTo(HaveOccurred())

			Expect(namespaceInode).To(Equal(fmt.Sprintf("%d", repoStat.Ino)))

			err = repo.Destroy(ns)
			Expect(err).NotTo(HaveOccurred())
		})

		It("logs the operation", func() {
			ns, err := repo.Create(name)
			Expect(err).NotTo(HaveOccurred())

			Expect(logger).To(gbytes.Say("create.created.*namespace.*test-ns"))

			err = repo.Destroy(ns)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not show up in ip netns list", func() {
			nsName := filepath.Base(repoDir)
			ns, err := repo.Create(nsName)
			Expect(err).NotTo(HaveOccurred())
			defer repo.Destroy(ns)

			output, err := exec.Command("ip", "netns", "list").CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
			Expect(output).NotTo(ContainSubstring(nsName))
		})

		Context("when the namespace file already exists", func() {
			var nsName string

			BeforeEach(func() {
				nsName = filepath.Base(repoDir)

				f, err := os.Create(filepath.Join(repoDir, nsName))
				Expect(err).NotTo(HaveOccurred())
				f.Close()
			})

			AfterEach(func() {
				os.RemoveAll(filepath.Join(repoDir, nsName))
			})

			It("returns ErrExist", func() {
				_, err := repo.Create(nsName)
				Expect(err).To(HaveOccurred())
				Expect(os.IsExist(err)).To(BeTrue())
			})

			It("logs the failure", func() {
				repo.Create(nsName)

				Expect(logger).To(gbytes.Say("create.bind-mount-failed"))
			})
		})
	})

	Describe("Get", func() {
		var repo namespace.Repository

		BeforeEach(func() {
			var err error
			repo, err = namespace.NewRepository(logger, repoDir, threadLocker)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns the correct namespace", func() {
			ns, err := repo.Create("some-namespace")
			Expect(err).NotTo(HaveOccurred())
			defer repo.Destroy(ns)

			nsFromGet, err := repo.Get("some-namespace")
			Expect(err).NotTo(HaveOccurred())

			Expect(ns.Name()).To(Equal(nsFromGet.Name()))

			err = nsFromGet.Execute(func(*os.File) error { return nil })
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the namespace file does not exist", func() {
			It("returns ErrNotExist", func() {
				_, err := repo.Get("test-ns")
				Expect(err).To(HaveOccurred())
				Expect(os.IsNotExist(err)).To(BeTrue())
			})

			It("logs the failure", func() {
				repo.Get("test-ns")

				Expect(logger).To(gbytes.Say("get.open-failed.*test-ns"))
			})
		})

		Context("when the namespace file exists", func() {
			var nsPath string

			BeforeEach(func() {
				var err error
				f, err := os.Create(filepath.Join(repoDir, "test-ns"))
				Expect(err).NotTo(HaveOccurred())

				nsPath = f.Name()
				Expect(f.Close()).To(Succeed())
			})

			It("returns the namespace", func() {
				ns, err := repo.Get("test-ns")
				Expect(err).NotTo(HaveOccurred())
				Expect(ns.Name()).To(Equal(nsPath))
			})

			It("keeps the file descriptor open", func() {
				ns, err := repo.Get("test-ns")
				Expect(err).NotTo(HaveOccurred())

				netns := ns.(*namespace.Netns)
				Expect(int(netns.Fd())).To(BeNumerically(">", 0))
			})

			It("logs the operation", func() {
				_, err := repo.Get("test-ns")
				Expect(err).NotTo(HaveOccurred())

				Expect(logger).To(gbytes.Say("get.complete.*namespace.*test-ns.*inode"))
			})
		})
	})
})
