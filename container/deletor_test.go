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
		deletor         container.Deletor
		executor        *fakes.Executor
		containerNS     namespace.Namespace
		namespaceOpener *fakes.Opener
	)

	BeforeEach(func() {
		executor = &fakes.Executor{}

		namespaceOpener = &fakes.Opener{}
		containerNS = &fakes.Namespace{NameStub: func() string { return "container ns sentinel" }}
		namespaceOpener.OpenPathReturns(containerNS, nil)

		deletor = container.Deletor{
			Executor:        executor,
			NamespaceOpener: namespaceOpener,
		}
	})

	It("should open the container namespace", func() {
		err := deletor.Delete("some-interface-name", "/path/to/container/namespace", "sandbox-name", "some-vxlan")
		Expect(err).NotTo(HaveOccurred())
		Expect(namespaceOpener.OpenPathCallCount()).To(Equal(1))
		Expect(namespaceOpener.OpenPathArgsForCall(0)).To(Equal("/path/to/container/namespace"))
	})

	Context("when opening the container namespace fails", func() {
		BeforeEach(func() {
			namespaceOpener.OpenPathReturns(nil, errors.New("POTATO"))
		})

		It("should return a meaningful error", func() {
			err := deletor.Delete("some-interface-name", "/path/to/container/namespace", "sandbox-name", "some-vxlan")
			Expect(err).To(MatchError("open container netns: POTATO"))
		})
	})

	It("should construct the correct command sequence", func() {
		err := deletor.Delete("some-interface-name", "/path/to/container/namespace", "sandbox-name", "some-vxlan")
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
					SandboxName:     "sandbox-name",
					VxlanDeviceName: "some-vxlan",
				},
			),
		))
	})

	Context("when executing fails", func() {
		BeforeEach(func() {
			executor.ExecuteReturns(errors.New("boom"))
		})

		It("should return the error", func() {
			err := deletor.Delete("some-interface-name", "/path/to/container/namespace", "sandbox-name", "some-vxlan")
			Expect(err).To(MatchError("boom"))
		})
	})
})
