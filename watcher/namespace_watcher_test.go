package watcher_test

import (
	"errors"
	"os"
	"syscall"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
	"github.com/tedsuo/ifrit"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NamespaceWatcher", func() {
	var (
		nsWatcher   *watcher.NamespaceWatcher
		missWatcher *fakes.MissWatcher
		ns          *fakes.Namespace
		process     ifrit.Process
	)

	BeforeEach(func() {
		missWatcher = &fakes.MissWatcher{}
		ns = &fakes.Namespace{}
		nsWatcher = &watcher.NamespaceWatcher{
			MissWatcher: missWatcher,
			Namespace:   ns,
			DeviceName:  "some-vxlan-name",
		}
	})

	JustBeforeEach(func() {
		process = ifrit.Invoke(nsWatcher)
	})

	AfterEach(func() {
		process.Signal(syscall.SIGINT)
	})

	It("calls StartMonitor with the right arguments", func() {
		Eventually(missWatcher.StartMonitorCallCount).Should(Equal(1))

		n, vxlanName := missWatcher.StartMonitorArgsForCall(0)
		Expect(n).To(Equal(ns))
		Expect(vxlanName).To(Equal("some-vxlan-name"))
	})

	Context("when StartMonitor fails", func() {
		BeforeEach(func() {
			missWatcher.StartMonitorReturns(errors.New("pineapple"))
		})

		It("returns a meaningful error", func() {
			errCh := process.Wait()
			Eventually(errCh).Should(Receive(MatchError("start monitor: pineapple")))
		})

		It("does not close the ready channel", func() {
			Consistently(process.Ready()).ShouldNot(BeClosed())
		})
	})

	It("calls StopMonitor when signaled", func() {
		Eventually(process.Ready()).Should(BeClosed())

		errCh := process.Wait()
		Consistently(errCh).ShouldNot(Receive())

		process.Signal(os.Kill)
		Eventually(missWatcher.StopMonitorCallCount).Should(Equal(1))
		Eventually(errCh).Should(Receive(BeNil()))

		namespace := missWatcher.StopMonitorArgsForCall(0)
		Expect(namespace).To(Equal(ns))
	})

	Context("when StopMonitor fails", func() {
		BeforeEach(func() {
			missWatcher.StopMonitorReturns(errors.New("kiwi"))
		})

		It("returns a meaningful error", func() {
			errCh := process.Wait()
			process.Signal(os.Kill)

			Eventually(errCh).Should(Receive(MatchError("stop monitor: kiwi")))
		})
	})
})
