package reloader_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/reloader"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reloader", func() {
	var (
		monitorReloader *reloader.Reloader
		watcher         *fakes.MissWatcher
		ns              *fakes.Namespace
	)

	BeforeEach(func() {
		watcher = &fakes.MissWatcher{}
		monitorReloader = &reloader.Reloader{
			Watcher: watcher,
		}
		ns = &fakes.Namespace{}

		ns.NameReturns("/some/sbox/path/vni-some-sandbox")
	})

	Describe("Callback", func() {
		It("restart the monitor for the given namespace", func() {
			err := monitorReloader.Callback(ns)
			Expect(err).NotTo(HaveOccurred())

			Expect(watcher.StartMonitorCallCount()).To(Equal(1))
			calledNS, vxlanDev := watcher.StartMonitorArgsForCall(0)
			Expect(calledNS).To(Equal(ns))
			Expect(vxlanDev).To(Equal("vxlansome-sandbox"))
		})

		Context("failure cases", func() {
			It("returns an error when the sandbox name is not valid", func() {
				ns.NameReturns("some-invalid-name")

				err := monitorReloader.Callback(ns)
				Expect(err).To(MatchError("get vxlan name: not a valid sandbox name"))
			})
			It("returns an error when monitor does not start", func() {
				watcher.StartMonitorReturns(errors.New("some-fake-error"))

				err := monitorReloader.Callback(ns)
				Expect(err).To(MatchError("start monitor: some-fake-error"))
			})
		})
	})
})
