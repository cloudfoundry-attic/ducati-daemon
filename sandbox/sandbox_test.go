package sandbox_test

import (
	"errors"

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
		invoker     *fakes.Invoker
		sbNamespace *fakes.Namespace
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		invoker = &fakes.Invoker{}
		sbNamespace = &fakes.Namespace{}

		sb = sandbox.New(logger, sbNamespace, invoker)
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
})
