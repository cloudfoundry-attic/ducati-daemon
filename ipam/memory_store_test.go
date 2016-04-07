package ipam_test

import (
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MemoryStore", func() {
	var store ipam.AllocatorStore
	var locker *fakes.Locker

	BeforeEach(func() {
		locker = &fakes.Locker{}
		store = ipam.NewStore(locker)
	})

	Describe("Lifecycle", func() {
		It("should allocate and release in a sensible way", func() {
			By("reserving a new ip")
			ok, err := store.Reserve("some-id", net.ParseIP("1.2.3.4"))
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())

			By("attempting to reserve the same ip")
			ok, err = store.Reserve("some-other-id", net.ParseIP("1.2.3.4"))
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())

			By("reserving another ip")
			ok, err = store.Reserve("some-other-id", net.ParseIP("1.2.3.5"))
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())

			By("releasing the first ip")
			err = store.ReleaseByID("some-id")
			Expect(err).NotTo(HaveOccurred())

			By("re-reserving the first ip")
			ok, err = store.Reserve("some-new-id", net.ParseIP("1.2.3.4"))
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("locks the data store when accessing and mutating", func() {
			store.Reserve("some-id", net.ParseIP("1.2.3.4"))
			Expect(locker.LockCallCount()).To(Equal(1))
			Expect(locker.UnlockCallCount()).To(Equal(1))

			store.ReleaseByID("some-id")
			Expect(locker.LockCallCount()).To(Equal(2))
			Expect(locker.UnlockCallCount()).To(Equal(2))
		})

		Context("when attempting to release an ID that is not allocated", func() {
			It("should silently succeed", func() {
				err := store.ReleaseByID("out-of-nowhere")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when multiple addresses are reserved for the saeme container", func() {
			It("releases them all", func() {
				By("reserving multiple IPs")
				ok, err := store.Reserve("some-id", net.ParseIP("1.2.3.4"))
				Expect(err).NotTo(HaveOccurred())
				Expect(ok).To(BeTrue())

				ok, err = store.Reserve("some-id", net.ParseIP("2.3.4.5"))
				Expect(err).NotTo(HaveOccurred())
				Expect(ok).To(BeTrue())

				By("releasing the id")
				err = store.ReleaseByID("some-id")
				Expect(err).NotTo(HaveOccurred())

				By("reserving the IPs again")
				ok, err = store.Reserve("some-id", net.ParseIP("1.2.3.4"))
				Expect(err).NotTo(HaveOccurred())
				Expect(ok).To(BeTrue())

				ok, err = store.Reserve("some-id", net.ParseIP("2.3.4.5"))
				Expect(err).NotTo(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
	})

	Describe("Contains", func() {
		It("tests whether a given container id has been allocated an ip", func() {
			ok, err := store.Reserve("some-id", net.ParseIP("1.2.3.4"))
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())

			Expect(store.Contains("some-id")).To(BeTrue())
			Expect(store.Contains("some-other-id")).To(BeFalse())
		})
	})
})
