package sandbox_test

import (
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/sandbox"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sandbox Repository", func() {
	var (
		repo     sandbox.Repository
		sandbox1 *sandbox.Sandbox
		sandbox2 *sandbox.Sandbox
		locker   *fakes.Locker
	)

	BeforeEach(func() {
		locker = &fakes.Locker{}
		repo = sandbox.NewRepository(locker)

		sandbox1 = &sandbox.Sandbox{}
		sandbox2 = &sandbox.Sandbox{}
	})

	Describe("Put", func() {
		It("adds the sandbox by name to the repo", func() {
			repo.Put("some-other-sandbox-name", sandbox1)

			sbox := repo.Get("some-other-sandbox-name")
			Expect(sbox).To(Equal(sandbox1))
		})

		It("locks and unlocks", func() {
			repo.Put("some-other-sandbox-name", sandbox1)
			Expect(locker.LockCallCount()).To(Equal(1))
			Expect(locker.UnlockCallCount()).To(Equal(1))
		})

		Context("when putting a different sandbox with the same key", func() {
			BeforeEach(func() {
				err := repo.Put("some-sandbox-name", sandbox1)
				Expect(err).NotTo(HaveOccurred())
			})

			It("overwrites that sandbox", func() {
				err := repo.Put("some-sandbox-name", sandbox2)
				Expect(err).To(MatchError(`sandbox "some-sandbox-name" already exists`))
			})

			It("locks and unlocks", func() {
				repo.Put("some-sandbox-name", sandbox2)

				Expect(locker.LockCallCount()).To(Equal(2))
				Expect(locker.UnlockCallCount()).To(Equal(2))
			})
		})
	})

	Describe("Get", func() {
		It("locks and unlocks", func() {
			repo.Get("some-other-sandbox-name")
			Expect(locker.LockCallCount()).To(Equal(1))
			Expect(locker.UnlockCallCount()).To(Equal(1))
		})

		Context("when getting a sandbox that hasn't been put", func() {
			It("returns nil", func() {
				sbox := repo.Get("unknown-sandbox")
				Expect(sbox).To(BeNil())
			})
		})

		Context("when getting a sandbox that has been put", func() {
			It("returns the sandbox", func() {
				repo.Put("some-sandbox-name", sandbox1)

				sbox := repo.Get("some-sandbox-name")
				Expect(sbox).To(Equal(sandbox1))
			})
		})
	})

	Describe("Remove", func() {
		BeforeEach(func() {
			repo.Put("some-sandbox-name", sandbox1)
			repo.Put("some-other-sandbox-name", sandbox2)
		})

		It("removes the sandbox by name", func() {
			repo.Remove("some-sandbox-name")

			sbox := repo.Get("some-sandbox-name")
			Expect(sbox).To(BeNil())
		})

		It("locks and unlocks", func() {
			repo.Remove("some-other-sandbox-name")
			Expect(locker.LockCallCount()).To(Equal(3))
			Expect(locker.UnlockCallCount()).To(Equal(3))
		})

		It("does not remove other sandbox", func() {
			repo.Remove("some-sandbox-name")

			sbox := repo.Get("some-other-sandbox-name")
			Expect(sbox).To(Equal(sandbox2))
		})
	})
})
