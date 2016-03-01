package container_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	cmd_fakes "github.com/cloudfoundry-incubator/ducati-daemon/commands/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	exec_fakes "github.com/cloudfoundry-incubator/ducati-daemon/executor/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delete", func() {
	var (
		deletor           container.Deletor
		executor          *exec_fakes.Executor
		sandboxRepoLocker *cmd_fakes.Locker
	)

	BeforeEach(func() {
		executor = &exec_fakes.Executor{}
		sandboxRepoLocker = &cmd_fakes.Locker{}
		deletor = container.Deletor{
			Executor: executor,
			Locker:   sandboxRepoLocker,
		}

	})

	It("should construct the correct command sequence", func() {
		deletorConfig := container.DeletorConfig{
			InterfaceName:   "some-interface-name",
			ContainerNSPath: "/some/container/namespace/path",
			SandboxNSPath:   "/some/sandbox/namespace/path",
		}

		err := deletor.Delete(deletorConfig)
		Expect(err).NotTo(HaveOccurred())

		Expect(executor.ExecuteCallCount()).To(Equal(1))

		Expect(executor.ExecuteArgsForCall(0)).To(Equal(
			commands.All(
				commands.InNamespace{
					Namespace: namespace.NewNamespace("/some/container/namespace/path"),
					Command: commands.DeleteLink{
						LinkName: "some-interface-name",
					},
				},

				commands.CleanupSandbox{
					Namespace: namespace.NewNamespace("/some/sandbox/namespace/path"),
					Locker:    sandboxRepoLocker,
				},
			),
		))
	})

	Context("when executing fails", func() {
		It("should return the error", func() {
			executor.ExecuteReturns(errors.New("boom"))

			deletorConfig := container.DeletorConfig{
				InterfaceName:   "some-interface-name",
				ContainerNSPath: "/some/container/namespace/path",
				SandboxNSPath:   "/some/sandbox/namespace/path",
			}

			err := deletor.Delete(deletorConfig)
			Expect(err).To(MatchError("boom"))
		})
	})

})
