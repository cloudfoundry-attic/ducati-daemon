package commands_test

import (
	"errors"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StartMonitor", func() {
	var (
		context           *fakes.Context
		sandboxRepository *fakes.Repository
		startMonitor      commands.StartMonitor
		fakeWatcher       *fakes.MissWatcher
		hostNS            *fakes.Namespace

		sandboxNS *fakes.Namespace
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		sandboxRepository = &fakes.Repository{}
		context.SandboxRepositoryReturns(sandboxRepository)

		sandboxNS = &fakes.Namespace{}
		sandboxRepository.GetReturns(sandboxNS, nil)

		hostNS = &fakes.Namespace{}

		fakeWatcher = &fakes.MissWatcher{}

		startMonitor = commands.StartMonitor{
			HostNamespace: hostNS,
			Watcher:       fakeWatcher,
			SandboxName:   "some-sandbox",
			VxlanLinkName: "some-vxlan-name",
		}

		hostNS.ExecuteStub = func(callback func(_ *os.File) error) error {
			return callback(nil)
		}
	})

	Describe("Execute", func() {
		It("it calls StartMonitor inside the HostNamespace", func() {

			hostNS.ExecuteStub = func(callback func(_ *os.File) error) error {
				Expect(fakeWatcher.StartMonitorCallCount()).To(Equal(0))
				callback(nil)
				Expect(fakeWatcher.StartMonitorCallCount()).To(Equal(1))
				return nil
			}

			err := startMonitor.Execute(context)
			Expect(err).NotTo(HaveOccurred())

			Expect(sandboxRepository.GetCallCount()).To(Equal(1))
			Expect(sandboxRepository.GetArgsForCall(0)).To(Equal("some-sandbox"))

			Expect(hostNS.ExecuteCallCount()).To(Equal(1))

			ns, linkName := fakeWatcher.StartMonitorArgsForCall(0)
			Expect(ns).To(Equal(sandboxNS))
			Expect(linkName).To(Equal("some-vxlan-name"))
		})

		Context("when getting the sandbox namespace fails", func() {
			BeforeEach(func() {
				sandboxRepository.GetReturns(nil, errors.New("potato"))
			})

			It("wraps and propogates the error", func() {
				err := startMonitor.Execute(context)
				Expect(err).To(MatchError("getting sandbox namespace: potato"))
			})
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
			Expect(startMonitor.String()).To(Equal("ip netns exec some-sandbox ip monitor neigh"))
		})
	})
})
