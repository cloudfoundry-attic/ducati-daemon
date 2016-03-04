package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	comm_fakes "github.com/cloudfoundry-incubator/ducati-daemon/commands/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StartMonitor", func() {
	var (
		context      *comm_fakes.Context
		startMonitor commands.StartMonitor
		fakeWatcher  *fakes.MissWatcher
		sandboxNS    *fakes.Namespace
	)

	BeforeEach(func() {
		context = &comm_fakes.Context{}
		fakeWatcher = &fakes.MissWatcher{}
		sandboxNS = &fakes.Namespace{}
		startMonitor = commands.StartMonitor{
			Watcher:   fakeWatcher,
			Namespace: sandboxNS,
		}

	})

	Describe("Execute", func() {
		It("calls out to watcher.StartMonitor", func() {
			err := startMonitor.Execute(context)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeWatcher.StartMonitorCallCount()).To(Equal(1))
			Expect(fakeWatcher.StartMonitorArgsForCall(0)).To(Equal(sandboxNS))
		})

		Context("when the StartMonitor call fails", func() {
			BeforeEach(func() {
				fakeWatcher.StartMonitorReturns(errors.New("banana"))
			})
			It("should wrap and return the error", func() {
				err := startMonitor.Execute(context)
				Expect(err).To(MatchError("watcher start monitor: banana"))
			})
		})
	})
})
