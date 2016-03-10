package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StartMonitor", func() {
	var (
		context      *fakes.Context
		startMonitor commands.StartMonitor
		fakeWatcher  *fakes.MissWatcher
		sandboxNS    *fakes.Namespace
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		fakeWatcher = &fakes.MissWatcher{}
		sandboxNS = &fakes.Namespace{}

		sandboxNS.NameReturns("some-namespace")

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

			It("wraps and propogates the error", func() {
				err := startMonitor.Execute(context)
				Expect(err).To(MatchError("watcher start monitor: banana"))
			})
		})
	})

	Describe("String", func() {
		It("describes itself", func() {
			Expect(startMonitor.String()).To(Equal("ip netns exec some-namespace ip monitor neigh"))
		})
	})
})
