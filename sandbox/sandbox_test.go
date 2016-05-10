package sandbox_test

import (
	"errors"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/sandbox"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Sandbox", func() {
	var (
		sb          sandbox.Sandbox
		logger      *lagertest.TestLogger
		sbNamespace *fakes.Namespace
		invoker     *fakes.Invoker
		watcher     *fakes.MissWatcher
		linkFactory *fakes.LinkFactory
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		invoker = &fakes.Invoker{}
		watcher = &fakes.MissWatcher{}
		linkFactory = &fakes.LinkFactory{}
		sbNamespace = &fakes.Namespace{}
		sbNamespace.ExecuteStub = func(callback func(*os.File) error) error {
			return callback(nil)
		}

		sb = sandbox.New(logger, sbNamespace, invoker, linkFactory, watcher)
	})

	Describe("Setup", func() {
		It("brings up the loopback adapter in the sandbox namespace", func() {
			sbNamespace.ExecuteStub = func(callback func(*os.File) error) error {
				Expect(linkFactory.SetUpCallCount()).To(Equal(0))
				err := callback(nil)
				Expect(linkFactory.SetUpCallCount()).To(Equal(1))
				return err
			}

			err := sb.Setup()
			Expect(err).NotTo(HaveOccurred())

			Expect(sbNamespace.ExecuteCallCount()).To(Equal(1))

			Expect(linkFactory.SetUpCallCount()).To(Equal(1))
			linkName := linkFactory.SetUpArgsForCall(0)
			Expect(linkName).To(Equal("lo"))
		})

		Context("when namespace execution fails", func() {
			BeforeEach(func() {
				sbNamespace.ExecuteReturns(errors.New("boysenberry"))
			})

			It("returns a meaningful error", func() {
				err := sb.Setup()
				Expect(err).To(MatchError("setup failed: boysenberry"))
			})
		})

		Context("when setting the link up fails", func() {
			BeforeEach(func() {
				linkFactory.SetUpReturns(errors.New("tomato"))
			})

			It("returns a meaningful error", func() {
				err := sb.Setup()
				Expect(err).To(MatchError("setup failed: set link up: tomato"))
			})
		})
	})

	Describe("Namespace", func() {
		It("returns the sandbox namespace", func() {
			ns := sb.Namespace()
			Expect(ns).To(Equal(sbNamespace))
		})
	})

	Describe("LaunchDNS", func() {
		var (
			runner  *fakes.Runner
			process *fakes.Process
			readyCh chan struct{}
			waitCh  chan error
		)

		BeforeEach(func() {
			runner = &fakes.Runner{}

			process = &fakes.Process{}
			invoker.InvokeReturns(process)

			readyCh = make(chan struct{}, 1)
			process.ReadyReturns(readyCh)

			waitCh = make(chan error, 1)
			process.WaitReturns(waitCh)
		})

		It("invokes the DNS runner", func() {
			err := sb.LaunchDNS(runner)
			Expect(err).NotTo(HaveOccurred())

			Expect(invoker.InvokeCallCount()).To(Equal(1))
			r := invoker.InvokeArgsForCall(0)
			Expect(r).To(Equal(runner))
		})

		Context("when the process exits before ready with an error", func() {
			BeforeEach(func() {
				waitCh <- errors.New("sprouts")
				close(waitCh)
			})

			It("return the exit error", func() {
				err := sb.LaunchDNS(runner)
				Expect(err).To(MatchError("launch dns: sprouts"))
			})
		})

		Context("when the process exits before ready without an error", func() {
			BeforeEach(func() {
				close(waitCh)
			})

			It("return the exit error", func() {
				err := sb.LaunchDNS(runner)
				Expect(err).To(MatchError("launch dns: unexpected server exit"))
			})
		})
	})

	Describe("VethDeviceCount", func() {
		BeforeEach(func() {
			linkFactory.VethDeviceCountReturns(99, nil)
		})

		It("returns gets veth device count from the sandbox", func() {
			sbNamespace.ExecuteStub = func(callback func(*os.File) error) error {
				Expect(linkFactory.VethDeviceCountCallCount()).To(Equal(0))
				err := callback(nil)
				Expect(linkFactory.VethDeviceCountCallCount()).To(Equal(1))
				return err
			}

			count, err := sb.VethDeviceCount()
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(99))

			Expect(sbNamespace.ExecuteCallCount()).To(Equal(1))
		})

		Context("when namespace execute fails", func() {
			BeforeEach(func() {
				sbNamespace.ExecuteReturns(errors.New("meatballs"))
			})

			It("returns a meaningful error", func() {
				_, err := sb.VethDeviceCount()
				Expect(err).To(MatchError("namespace execute: meatballs"))
			})
		})

		Context("when the link factory fails to return the veth device count", func() {
			BeforeEach(func() {
				linkFactory.VethDeviceCountReturns(0, errors.New("tomato"))
			})

			It("returns a meaningful error", func() {
				_, err := sb.VethDeviceCount()
				Expect(err).To(MatchError("veth device count: tomato"))
			})
		})
	})

	Describe("Teardown", func() {
		It("tears down the sandbox", func() {
			err := sb.Teardown()
			Expect(err).NotTo(HaveOccurred())
		})

		It("stops the arp miss monitor", func() {
			err := sb.Teardown()
			Expect(err).NotTo(HaveOccurred())

			Expect(watcher.StopMonitorCallCount()).To(Equal(1))
			targetNS := watcher.StopMonitorArgsForCall(0)
			Expect(targetNS).To(Equal(sbNamespace))
		})

		Context("when the sandbox has already been torn down", func() {
			BeforeEach(func() {
				err := sb.Teardown()
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an AlreadyDestroyedError", func() {
				err := sb.Teardown()
				Expect(err).To(Equal(sandbox.AlreadyDestroyedError))
			})

			It("does NOT try to stop the monitor AGAIN", func() {
				err := sb.Teardown()
				Expect(err).To(HaveOccurred())

				Expect(watcher.StopMonitorCallCount()).To(Equal(1))
			})
		})

		Context("when there is an error stopping the miss watcher", func() {
			It("returns a meaningful error", func() {
				watcher.StopMonitorReturns(errors.New("spaghetti"))

				err := sb.Teardown()
				Expect(err).To(MatchError("stop monitor: spaghetti"))
			})
		})
	})
})
