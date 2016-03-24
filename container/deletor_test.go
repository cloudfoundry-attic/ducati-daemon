package container_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delete", func() {
	var (
		deletor           container.Deletor
		executor          *fakes.Executor
		sandboxRepoLocker *fakes.NamedLocker
		watcher           *fakes.MissWatcher
		sandboxRepository *fakes.Repository
		containerNS       namespace.Namespace
		sandboxNS         *fakes.Namespace
		namespaceOpener   *fakes.Opener

		deletorConfig container.DeletorConfig
	)

	BeforeEach(func() {
		executor = &fakes.Executor{}
		sandboxRepoLocker = &fakes.NamedLocker{}
		watcher = &fakes.MissWatcher{}
		sandboxRepository = &fakes.Repository{}
		containerNS = &fakes.Namespace{NameStub: func() string { return "container ns sentinel" }}
		namespaceOpener = &fakes.Opener{}
		namespaceOpener.OpenPathReturns(containerNS, nil)
		sandboxNS = &fakes.Namespace{NameStub: func() string { return "sandbox ns sentinel" }}
		deletor = container.Deletor{
			Executor:          executor,
			SandboxRepository: sandboxRepository,
			NamedLocker:       sandboxRepoLocker,
			Watcher:           watcher,
			NamespaceOpener:   namespaceOpener,
		}

		deletorConfig = container.DeletorConfig{
			InterfaceName:   "some-interface-name",
			ContainerNSPath: "/some/container/ns/path",
			SandboxNS:       sandboxNS,
			VxlanDeviceName: "some-vxlan",
		}
	})

	It("should open the container namespace", func() {
		err := deletor.Delete(deletorConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(namespaceOpener.OpenPathCallCount()).To(Equal(1))
		Expect(namespaceOpener.OpenPathArgsForCall(0)).To(Equal("/some/container/ns/path"))
	})

	Context("when opening the container namespace fails", func() {
		It("should return a meaningful error", func() {
			namespaceOpener.OpenPathReturns(nil, errors.New("POTATO"))

			err := deletor.Delete(deletorConfig)
			Expect(err).To(MatchError("open container netns: POTATO"))
		})
	})

	It("should construct the correct command sequence", func() {
		err := deletor.Delete(deletorConfig)
		Expect(err).NotTo(HaveOccurred())

		Expect(executor.ExecuteCallCount()).To(Equal(1))

		Expect(executor.ExecuteArgsForCall(0)).To(Equal(
			commands.All(
				commands.InNamespace{
					Namespace: containerNS,
					Command: commands.DeleteLink{
						LinkName: "some-interface-name",
					},
				},

				commands.CleanupSandbox{
					Namespace:         sandboxNS,
					SandboxRepository: sandboxRepository,
					NamedLocker:       sandboxRepoLocker,
					Watcher:           watcher,
					VxlanDeviceName:   "some-vxlan",
				},
			),
		))
	})

	Context("when executing fails", func() {
		It("should return the error", func() {
			executor.ExecuteReturns(errors.New("boom"))

			err := deletor.Delete(deletorConfig)
			Expect(err).To(MatchError("boom"))
		})
	})
})
