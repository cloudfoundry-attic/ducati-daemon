package commands_test

import (
	"errors"
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/sandbox"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CleanupSandbox", func() {
	var (
		context               *fakes.Context
		sbox                  *fakes.Sandbox
		sandboxNS             *fakes.Namespace
		linkFactory           *fakes.LinkFactory
		cleanupSandboxCommand commands.CleanupSandbox
		missWatcher           *fakes.MissWatcher
		namespaceRepository   *fakes.Repository
		sandboxRepo           *fakes.SandboxRepository
	)

	BeforeEach(func() {
		context = &fakes.Context{}

		sandboxNS = &fakes.Namespace{}
		sandboxNS.NameReturns("sandbox-name")

		sbox = &fakes.Sandbox{}
		sbox.NamespaceReturns(sandboxNS)

		linkFactory = &fakes.LinkFactory{}
		context.LinkFactoryReturns(linkFactory)
		missWatcher = &fakes.MissWatcher{}
		namespaceRepository = &fakes.Repository{}

		sandboxRepo = &fakes.SandboxRepository{}
		sandboxRepo.GetStub = func(key string) (sandbox.Sandbox, error) {
			if key == "sandbox-name" {
				return sbox, nil
			}
			return nil, sandbox.NotFoundError
		}

		context.SandboxRepositoryReturns(sandboxRepo)
		context.SandboxNamespaceRepositoryReturns(namespaceRepository)

		cleanupSandboxCommand = commands.CleanupSandbox{
			SandboxName:     "sandbox-name",
			Watcher:         missWatcher,
			VxlanDeviceName: "some-vxlan",
		}

		sandboxNS.ExecuteStub = func(callback func(ns *os.File) error) error {
			err := callback(nil)
			if err != nil {
				return fmt.Errorf("callback failed: %s", err)
			}
			return nil
		}
	})

	It("gets the sandbox by name", func() {
		err := cleanupSandboxCommand.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(sandboxRepo.GetCallCount()).To(Equal(1))
		Expect(sandboxRepo.GetArgsForCall(0)).To(Equal("sandbox-name"))
	})

	Context("when sandbox doesn't exist", func() {
		It("returns an error", func() {
			sandboxRepo.GetReturns(nil, sandbox.NotFoundError)

			err := cleanupSandboxCommand.Execute(context)
			Expect(err).To(MatchError("get sandbox: not found"))
		})
	})

	It("locks and unlocks on the namespace", func() {
		err := cleanupSandboxCommand.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(sbox.LockCallCount()).To(Equal(1))
		Expect(sbox.UnlockCallCount()).To(Equal(1))
	})

	It("counts the veth devices inside the sandbox", func() {
		sandboxNS.ExecuteStub = func(callback func(ns *os.File) error) error {
			Expect(linkFactory.VethDeviceCountCallCount()).To(Equal(0))
			callback(nil)
			Expect(linkFactory.VethDeviceCountCallCount()).To(Equal(1))
			return nil
		}

		Expect(cleanupSandboxCommand.Execute(context)).To(Succeed())
		Expect(sandboxNS.ExecuteCallCount()).To(Equal(1))
	})

	Context("when counting the veth devices fails", func() {
		BeforeEach(func() {
			linkFactory.VethDeviceCountReturns(0, errors.New("some error"))
		})

		It("wraps and returns an error", func() {
			err := cleanupSandboxCommand.Execute(context)
			Expect(err).To(MatchError("in namespace sandbox-name: callback failed: counting veth devices: some error"))
		})
	})

	Context("when there is STILL a veth device in the sandbox", func() {
		BeforeEach(func() {
			linkFactory.VethDeviceCountReturns(1, nil)
		})

		It("does NOT destroy the namespace", func() {
			err := cleanupSandboxCommand.Execute(context)
			Expect(err).NotTo(HaveOccurred())

			Expect(namespaceRepository.DestroyCallCount()).To(Equal(0))
		})

		It("does NOT destroy the vxlan device", func() {
			err := cleanupSandboxCommand.Execute(context)
			Expect(err).NotTo(HaveOccurred())

			Expect(linkFactory.DeleteLinkByNameCallCount()).To(Equal(0))
		})
	})

	Context("when there are no more veth devices in the sandbox", func() {
		It("stops monitoring the namespace", func() {
			err := cleanupSandboxCommand.Execute(context)
			Expect(err).NotTo(HaveOccurred())

			Expect(missWatcher.StopMonitorCallCount()).To(Equal(1))
			Expect(missWatcher.StopMonitorArgsForCall(0)).To(Equal(sandboxNS))
		})

		Context("when stopping monitoring fails", func() {
			BeforeEach(func() {
				missWatcher.StopMonitorReturns(errors.New("potato"))
			})

			It("wraps and returns the error", func() {
				err := cleanupSandboxCommand.Execute(context)
				Expect(err).To(MatchError("watcher stop monitor: potato"))
			})

			It("does not attempt to destroy the namespace", func() {
				cleanupSandboxCommand.Execute(context)

				Expect(namespaceRepository.DestroyCallCount()).To(Equal(0))
			})
		})

		It("destroys the vxlan device in the sandbox namespace", func() {
			sandboxNS.ExecuteStub = func(callback func(ns *os.File) error) error {
				Expect(linkFactory.DeleteLinkByNameCallCount()).To(Equal(0))
				callback(nil)
				Expect(linkFactory.DeleteLinkByNameCallCount()).To(Equal(1))
				Expect(linkFactory.DeleteLinkByNameArgsForCall(0)).To(Equal(cleanupSandboxCommand.VxlanDeviceName))
				return nil
			}

			err := cleanupSandboxCommand.Execute(context)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when there is an error destroying vxlan device", func() {
			BeforeEach(func() {
				linkFactory.DeleteLinkByNameReturns(errors.New("some-error"))
			})

			It("checks if the link still exists", func() {
				cleanupSandboxCommand.Execute(context)
				Expect(linkFactory.ExistsCallCount()).To(Equal(1))

				linkName := linkFactory.ExistsArgsForCall(0)
				Expect(linkName).To(Equal(cleanupSandboxCommand.VxlanDeviceName))
			})

			Context("when the link no longer exists", func() {
				BeforeEach(func() {
					linkFactory.ExistsReturns(false)
				})

				It("returns without an error", func() {
					err := cleanupSandboxCommand.Execute(context)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("when the link still exists", func() {
				BeforeEach(func() {
					linkFactory.ExistsReturns(true)
				})

				It("wraps and returns the original error", func() {
					err := cleanupSandboxCommand.Execute(context)
					Expect(err).To(MatchError("in namespace sandbox-name: callback failed: destroying vxlan some-vxlan: some-error"))
				})
			})
		})

		It("destroys the namespace", func() {
			err := cleanupSandboxCommand.Execute(context)
			Expect(err).NotTo(HaveOccurred())

			Expect(namespaceRepository.DestroyCallCount()).To(Equal(1))
			Expect(namespaceRepository.DestroyArgsForCall(0)).To(Equal(sandboxNS))
		})

		Context("when theres an error destroying namespace", func() {
			BeforeEach(func() {
				namespaceRepository.DestroyReturns(errors.New("some-destroy-error"))
			})

			It("wraps and propogates the error", func() {
				Expect(cleanupSandboxCommand.Execute(context)).To(MatchError("destroying sandbox sandbox-name: some-destroy-error"))
			})
		})

		It("removes the sandbox from the repo", func() {
			err := cleanupSandboxCommand.Execute(context)
			Expect(err).NotTo(HaveOccurred())

			Expect(sandboxRepo.RemoveCallCount()).To(Equal(1))
			Expect(sandboxRepo.RemoveArgsForCall(0)).To(Equal("sandbox-name"))
		})
	})

	Describe("String", func() {
		It("describes itself", func() {
			Expect(cleanupSandboxCommand.String()).To(Equal("cleanup-sandbox sandbox-name"))
		})
	})
})
