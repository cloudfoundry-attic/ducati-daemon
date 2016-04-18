package sandbox_test

import (
	"os"
	"syscall"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/sandbox"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("Sandbox", func() {
	var (
		sb        *sandbox.Sandbox
		invoker   *fakes.Invoker
		nsWatcher *fakes.Runner
		nsProcess *fakes.Process
	)

	BeforeEach(func() {
		nsWatcher = &fakes.Runner{}
		nsProcess = &fakes.Process{}

		invoker = &fakes.Invoker{}
		invoker.InvokeStub = func(runner ifrit.Runner) ifrit.Process {
			return nsProcess
		}

		sb = &sandbox.Sandbox{
			Invoker:          invoker,
			NamespaceWatcher: nsWatcher,
		}
	})

	Describe("Run", func() {
		var process ifrit.Process

		AfterEach(func() {
			if process != nil {
				process.Signal(syscall.SIGINT)
			}
		})

		It("closes the ready channel after setting up", func() {
			process = ifrit.Invoke(sb)
			Eventually(process.Ready()).Should(BeClosed())
		})

		It("invokes the NamespaceWatcher", func() {
			process = ifrit.Invoke(sb)
			Eventually(invoker.InvokeCallCount).Should(Equal(1))
			runner := invoker.InvokeArgsForCall(0)
			Expect(runner).To(Equal(nsWatcher))
		})

		It("doesn't return until it is signaled", func() {
			process = ifrit.Invoke(sb)
			errCh := process.Wait()
			Consistently(errCh).ShouldNot(Receive())

			process.Signal(os.Kill)
			Eventually(errCh).Should(Receive(BeNil()))
		})

		It("propagates the signal to the namespace watcher", func() {
			process = ifrit.Invoke(sb)
			errCh := process.Wait()
			Consistently(errCh).ShouldNot(Receive())

			process.Signal(os.Kill)

			Eventually(nsProcess.SignalCallCount).Should(Equal(1))
			signal := nsProcess.SignalArgsForCall(0)
			Expect(signal).To(Equal(os.Kill))
		})
	})
})
