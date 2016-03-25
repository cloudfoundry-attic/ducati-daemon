package namespace_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Namespace", func() {
	Describe("Name", func() {
		It("returns the file Name()", func() {
			tempFile, err := ioutil.TempFile("", "whatever")
			Expect(err).NotTo(HaveOccurred())
			tempFile.Close()
			defer os.Remove(tempFile.Name())

			actualName := tempFile.Name()
			Expect(namespace.Netns{File: tempFile}.Name()).To(Equal(actualName))
		})
	})

	Describe("Execute", func() {
		var nsInode uint64

		BeforeEach(func() {
			err := exec.Command("ip", "netns", "add", "ns-test-ns").Run()
			Expect(err).NotTo(HaveOccurred())

			var stat unix.Stat_t
			err = unix.Stat("/var/run/netns/ns-test-ns", &stat)
			Expect(err).NotTo(HaveOccurred())

			nsInode = stat.Ino
		})

		AfterEach(func() {
			err := exec.Command("ip", "netns", "delete", "ns-test-ns").Run()
			Expect(err).NotTo(HaveOccurred())
		})

		It("runs the closure in the namespace", func() {
			nsFile, err := os.Open("/var/run/netns/ns-test-ns")
			Expect(err).NotTo(HaveOccurred())
			ns := namespace.Netns{File: nsFile}

			var namespaceInode string
			closure := func(f *os.File) error {
				// Stat of "/proc/self/ns/net" flakey due to fs caching
				output, err := exec.Command("stat", "-L", "-c", "%i", "/proc/self/ns/net").CombinedOutput()
				namespaceInode = strings.TrimSpace(string(output))
				return err
			}

			err = ns.Execute(closure)
			Expect(err).NotTo(HaveOccurred())
			Expect(namespaceInode).To(Equal(fmt.Sprintf("%d", nsInode)))
		})
	})

	Describe("OpenPath", func() {
		var (
			opener   *namespace.PathOpener
			tempFile *os.File
		)

		BeforeEach(func() {
			opener = &namespace.PathOpener{}

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
})
