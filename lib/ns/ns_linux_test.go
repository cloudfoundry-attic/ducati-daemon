package ns_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/ns"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Linux network namespace", func() {
	var namespacer ns.Namespacer

	BeforeEach(func() {
		namespacer = ns.LinuxNamespacer
	})

	Describe("GetFromPath", func() {
		It("opens the path and returns a namespace handle", func() {
			path := "/proc/self/ns/net"

			handle, err := namespacer.GetFromPath(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(handle).NotTo(BeNil())
			Expect(handle.IsOpen()).To(BeTrue())

			handleFile := os.NewFile(handle.Fd(), path)
			handleFileInfo, err := handleFile.Stat()
			Expect(err).NotTo(HaveOccurred())

			file, err := os.Open(path)
			Expect(err).NotTo(HaveOccurred())

			fileInfo, err := file.Stat()
			Expect(err).NotTo(HaveOccurred())

			Expect(os.SameFile(handleFileInfo, fileInfo)).To(BeTrue())
		})

		Context("when the namespace path cannot be opened", func() {
			It("returns a helpful error", func() {
				_, err := namespacer.GetFromPath("")
				Expect(err).To(MatchError("open failed: no such file or directory"))
			})
		})
	})

	Describe("Set", func() {
		BeforeEach(func() {
			cmd := exec.Command("ip", "netns", "add", "some-other-ns")
			Expect(cmd.Run()).To(Succeed())
		})

		AfterEach(func() {
			cmd := exec.Command("ip", "netns", "del", "some-other-ns")
			Expect(cmd.Run()).To(Succeed())
		})

		It("sets the namespace to the handle passed in", func() {
			selfNS := "/proc/self/ns/net"
			newNS := "/var/run/netns/some-other-ns"

			parentInode, err := exec.Command("stat", "-L", "-c", "%i", selfNS).CombinedOutput()
			newNSInode, err := exec.Command("stat", "-L", "-c", "%i", newNS).CombinedOutput()

			Expect(parentInode).NotTo(Equal(newNSInode))

			originalNSHandle, err := namespacer.GetFromPath(selfNS)
			Expect(err).NotTo(HaveOccurred())
			// defer newNSHandle.Close()

			newNSHandle, err := namespacer.GetFromPath(newNS)
			Expect(err).NotTo(HaveOccurred())
			// defer newNSHandle.Close()

			Expect(namespacer.Set(newNSHandle)).To(Succeed())
			selfInodeInNewNS, err := exec.Command("stat", "-L", "-c", "%i", selfNS).CombinedOutput()
			Expect(namespacer.Set(originalNSHandle)).To(Succeed())

			Expect(newNSInode).To(Equal(selfInodeInNewNS))
		})

		Context("when the netns cannot be set", func() {
			var nonExistantNS ns.Handle
			var tempDir string

			BeforeEach(func() {
				var err error
				tempDir, err = ioutil.TempDir("", "")
				Expect(err).NotTo(HaveOccurred())

				nonExistantNS, err = namespacer.GetFromPath(tempDir)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				os.RemoveAll(tempDir)
			})

			It("returns a helpful error message", func() {
				err := namespacer.Set(nonExistantNS)
				Expect(err).To(MatchError("failed to set namespace: invalid argument"))
			})
		})
	})

	Describe("Handle", func() {
		Describe("Close", func() {
			It("closes the file", func() {
				tempDir, err := ioutil.TempDir("", "")
				Expect(err).NotTo(HaveOccurred())

				path := filepath.Join(tempDir, "file.txt")
				err = ioutil.WriteFile(path, []byte("hello, my friends"), 0644)
				Expect(err).NotTo(HaveOccurred())

				fileHandle, err := namespacer.GetFromPath(path)
				Expect(err).NotTo(HaveOccurred())

				file := os.NewFile(fileHandle.Fd(), path)
				Expect(fileHandle.Close()).To(Succeed())

				data := make([]byte, 3)
				_, err = file.Read(data)
				Expect(err).To(HaveOccurred())
			})

			It("can check that the file is closed", func() {
				handle, err := namespacer.GetFromPath("/")
				Expect(err).NotTo(HaveOccurred())

				Expect(handle.Close()).To(Succeed())
				Expect(handle.IsOpen()).To(BeFalse())
			})

			Context("when close fails", func() {
				It("returns a helpful error message", func() {
					handle, err := namespacer.GetFromPath("/proc/self/ns/net")
					Expect(err).NotTo(HaveOccurred())
					Expect(handle.Close()).To(Succeed())

					err = handle.Close()
					Expect(err).To(MatchError("close failed: bad file descriptor"))
				})
			})
		})
	})
})
