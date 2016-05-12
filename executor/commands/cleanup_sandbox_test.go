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
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("CleanupSandbox", func() {
	var (
		context               *fakes.Context
		logger                *lagertest.TestLogger
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

		logger = lagertest.NewTestLogger("test")
		context.LoggerReturns(logger)

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
		It("returns without an error", func() {
			sandboxRepo.GetReturns(nil, sandbox.NotFoundError)

			err := cleanupSandboxCommand.Execute(context)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when getting the sandbox fails", func() {
		BeforeEach(func() {
			sandboxRepo.GetReturns(nil, errors.New("raisins"))
		})

		It("returns an error", func() {
			err := cleanupSandboxCommand.Execute(context)
			Expect(err).To(MatchError("get sandbox: raisins"))
		})

		It("logs the error", func() {
			cleanupSandboxCommand.Execute(context)

			Expect(logger).To(gbytes.Say("get-sandbox-failed.*raisins"))
		})
	})

	It("locks and unlocks on the sandbox", func() {
		err := cleanupSandboxCommand.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(sbox.LockCallCount()).To(Equal(1))
		Expect(sbox.UnlockCallCount()).To(Equal(1))
	})

	It("counts the veth devices inside the sandbox", func() {
		err := cleanupSandboxCommand.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(sbox.VethDeviceCountCallCount()).To(Equal(1))
	})

	Context("when counting the veth devices fails", func() {
		BeforeEach(func() {
			sbox.VethDeviceCountReturns(0, errors.New("some error"))
		})

		It("wraps and returns an error", func() {
			err := cleanupSandboxCommand.Execute(context)
			Expect(err).To(MatchError("counting veth devices: some error"))
		})
	})

	Context("when there is STILL a veth device in the sandbox", func() {
		BeforeEach(func() {
			sbox.VethDeviceCountReturns(1, nil)
		})

		It("does NOT destroy the namespace", func() {
			err := cleanupSandboxCommand.Execute(context)
			Expect(err).NotTo(HaveOccurred())

			Expect(sandboxRepo.DestroyCallCount()).To(Equal(0))
		})

		It("does NOT destroy the vxlan device", func() {
			err := cleanupSandboxCommand.Execute(context)
			Expect(err).NotTo(HaveOccurred())

			Expect(linkFactory.DeleteLinkByNameCallCount()).To(Equal(0))
		})

		It("does NOT destroy the sandbox", func() {
			err := cleanupSandboxCommand.Execute(context)
			Expect(err).NotTo(HaveOccurred())

			Expect(sandboxRepo.DestroyCallCount()).To(Equal(0))
		})
	})

	Context("when there are no veth devices in the sandbox", func() {
		It("removes the vxlan device in the sandbox namespace", func() {
			sandboxNS.ExecuteStub = func(callback func(ns *os.File) error) error {
				Expect(linkFactory.DeleteLinkByNameCallCount()).To(Equal(0))
				err := callback(nil)
				Expect(linkFactory.DeleteLinkByNameCallCount()).To(Equal(1))
				return err
			}

			err := cleanupSandboxCommand.Execute(context)
			Expect(err).NotTo(HaveOccurred())

			Expect(sandboxNS.ExecuteCallCount()).To(Equal(1))
			Expect(linkFactory.DeleteLinkByNameArgsForCall(0)).To(Equal("some-vxlan"))
		})

		It("destroys the sandbox", func() {
			err := cleanupSandboxCommand.Execute(context)
			Expect(err).NotTo(HaveOccurred())

			Expect(sandboxRepo.DestroyCallCount()).To(Equal(1))
			Expect(sandboxRepo.DestroyArgsForCall(0)).To(Equal("sandbox-name"))
		})

		Context("when destroy fails with AlreadyDestroyedError", func() {
			BeforeEach(func() {
				sandboxRepo.DestroyReturns(sandbox.AlreadyDestroyedError)
			})

			It("does not fail", func() {
				err := cleanupSandboxCommand.Execute(context)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when destroy fails with a NotFoundError", func() {
			BeforeEach(func() {
				sandboxRepo.DestroyReturns(sandbox.NotFoundError)
			})

			It("does not fail", func() {
				err := cleanupSandboxCommand.Execute(context)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when destroy fails", func() {
			BeforeEach(func() {
				sandboxRepo.DestroyReturns(errors.New("potato"))
			})

			It("wraps and returns the error", func() {
				err := cleanupSandboxCommand.Execute(context)
				Expect(err).To(MatchError("sandbox destroy: potato"))
			})
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
	})

	Describe("String", func() {
		It("describes itself", func() {
			Expect(cleanupSandboxCommand.String()).To(Equal("cleanup-sandbox sandbox-name"))
		})
	})
})
