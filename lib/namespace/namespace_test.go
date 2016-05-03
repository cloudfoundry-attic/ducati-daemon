package namespace_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Namespace", func() {
	var logger *lagertest.TestLogger

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
	})

	Describe("Name", func() {
		It("returns the file Name()", func() {
			tempFile, err := ioutil.TempFile("", "whatever")
			Expect(err).NotTo(HaveOccurred())
			tempFile.Close()
			defer os.Remove(tempFile.Name())

			actualName := tempFile.Name()
			Expect(namespace.Netns{File: tempFile, Logger: logger}.Name()).To(Equal(actualName))
		})
	})

	Describe("Execute", func() {
		var (
			nsInode uint64
			ns      *namespace.Netns
			nsName  string
			nsPath  string
		)

		BeforeEach(func() {
			nsName = fmt.Sprintf("ns-test-ns-%d", GinkgoParallelNode())
			nsPath = fmt.Sprintf("/var/run/netns/%s", nsName)
			err := exec.Command("ip", "netns", "add", nsName).Run()
			Expect(err).NotTo(HaveOccurred())

			var stat unix.Stat_t
			err = unix.Stat(nsPath, &stat)
			Expect(err).NotTo(HaveOccurred())

			nsInode = stat.Ino

			nsFile, err := os.Open(nsPath)
			Expect(err).NotTo(HaveOccurred())
			ns = &namespace.Netns{File: nsFile, Logger: logger}
		})

		AfterEach(func() {
			err := exec.Command("ip", "netns", "delete", nsName).Run()
			Expect(err).NotTo(HaveOccurred())
		})

		It("runs the closure in the namespace", func() {
			var namespaceInode string
			closure := func(f *os.File) error {
				// Stat of "/proc/self/ns/net" flakey due to fs caching
				output, err := exec.Command("stat", "-L", "-c", "%i", "/proc/self/ns/net").CombinedOutput()
				namespaceInode = strings.TrimSpace(string(output))
				return err
			}

			err := ns.Execute(closure)
			Expect(err).NotTo(HaveOccurred())
			Expect(namespaceInode).To(Equal(fmt.Sprintf("%d", nsInode)))
		})

		It("logs the operation and namespace", func() {
			err := ns.Execute(func(*os.File) error { return nil })
			Expect(err).NotTo(HaveOccurred())

			Expect(logger).To(gbytes.Say("execute.callback-invoked.*ns-test-ns.*inode"))
			Expect(logger).To(gbytes.Say("execute.callback-complete.*ns-test-ns.*inode"))
		})

		Context("when the callback fails", func() {
			It("logs the error", func() {
				ns.Execute(func(*os.File) error { return errors.New("potato") })

				Expect(logger).To(gbytes.Say("execute.callback-failed.*potato"))
			})
		})
	})

	Describe("OpenPath", func() {
		var (
			opener   *namespace.PathOpener
			tempFile *os.File
		)

		BeforeEach(func() {
			opener = &namespace.PathOpener{
				Logger: logger,
			}

			var err error
			tempFile, err = ioutil.TempFile("", "OpenPath")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			tempFile.Close()
			os.Remove(tempFile.Name())
		})

		It("opens the specified file and returns it as a Namespace", func() {
			ns, err := opener.OpenPath(tempFile.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(ns.Name()).To(Equal(tempFile.Name()))

			netns, ok := ns.(*namespace.Netns)
			Expect(ok).To(BeTrue())
			Expect(int(netns.Fd())).To(BeNumerically(">", 0))
			Expect(netns.Logger).To(Equal(logger))
		})

		Context("when the file cannot be opened", func() {
			BeforeEach(func() {
				tempFile.Close()
				os.Remove(tempFile.Name())
			})

			It("returns a meaningful error", func() {
				_, err := opener.OpenPath(tempFile.Name())
				Expect(err).To(MatchError(HavePrefix(fmt.Sprintf("open %s:", tempFile.Name()))))
			})
		})
	})

	Describe("Fd", func() {
		It("returns the file descriptor of the open namespace", func() {
			opener := &namespace.PathOpener{}
			temp, err := ioutil.TempFile("", "whatever")
			Expect(err).NotTo(HaveOccurred())
			temp.Close()
			defer os.Remove(temp.Name())

			ns, err := opener.OpenPath(temp.Name())
			Expect(err).NotTo(HaveOccurred())

			netns, ok := ns.(*namespace.Netns)
			Expect(ok).To(BeTrue())

			Expect(ns.Fd()).To(Equal(netns.Fd()))
			Expect(int(ns.Fd())).To(BeNumerically(">", 0))
		})
	})

	Describe("MarsalJSON", func() {
		It("marshals as a name and inode", func() {
			tempFile, err := ioutil.TempFile("", "whatever")
			Expect(err).NotTo(HaveOccurred())
			defer tempFile.Close()
			defer os.Remove(tempFile.Name())

			actualName := tempFile.Name()

			var stat unix.Stat_t
			err = unix.Stat(actualName, &stat)
			Expect(err).NotTo(HaveOccurred())

			ns := &namespace.Netns{File: tempFile, Logger: logger}
			expectedJSON := fmt.Sprintf(`{ "name": "%s", "inode": "%d" }`, actualName, stat.Ino)

			json, err := json.Marshal(ns)
			Expect(err).NotTo(HaveOccurred())

			Expect(json).To(MatchJSON(expectedJSON))
		})
	})

	Describe("String", func() {
		It("returns the name and the inode of the underlying namespace file", func() {
			tempFile, err := ioutil.TempFile("", "whatever")
			Expect(err).NotTo(HaveOccurred())
			defer tempFile.Close()
			defer os.Remove(tempFile.Name())

			actualName := tempFile.Name()

			var stat unix.Stat_t
			err = unix.Stat(actualName, &stat)
			Expect(err).NotTo(HaveOccurred())

			ns := &namespace.Netns{File: tempFile, Logger: logger}

			Expect(ns.String()).To(Equal(fmt.Sprintf("%s:[%d]", actualName, stat.Ino)))
		})
	})
})
