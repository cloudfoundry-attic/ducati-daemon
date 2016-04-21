package sandbox_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/sandbox"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Sandbox Repository", func() {
	var (
		logger        *lagertest.TestLogger
		locker        *fakes.Locker
		sboxNamespace *fakes.Namespace
		namespaceRepo *fakes.Repository
		invoker       *fakes.Invoker
		sandboxRepo   sandbox.Repository
		linkFactory   *fakes.LinkFactory
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		locker = &fakes.Locker{}
		invoker = &fakes.Invoker{}
		sboxNamespace = &fakes.Namespace{}
		namespaceRepo = &fakes.Repository{}
		namespaceRepo.CreateReturns(sboxNamespace, nil)
		linkFactory = &fakes.LinkFactory{}
		sandboxRepo = sandbox.NewRepository(
			logger,
			locker,
			namespaceRepo,
			invoker,
			linkFactory,
		)
	})

	Describe("Create", func() {
		It("returns the created sandbox", func() {
			sbox, err := sandboxRepo.Create("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())
			Expect(sbox).NotTo(BeNil())
		})

		It("locks and unlocks", func() {
			_, err := sandboxRepo.Create("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(locker.LockCallCount()).To(Equal(1))
			Expect(locker.UnlockCallCount()).To(Equal(1))
		})

		It("creates a namespace", func() {
			_, err := sandboxRepo.Create("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(namespaceRepo.CreateCallCount()).To(Equal(1))
			nsName := namespaceRepo.CreateArgsForCall(0)
			Expect(nsName).To(Equal("some-sandbox-name"))
		})

		It("populates the namespace on the sandbox", func() {
			sbox, err := sandboxRepo.Create("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(sbox.Namespace()).To(Equal(sboxNamespace))
		})

		Context("when creating the namespace fails", func() {
			BeforeEach(func() {
				namespaceRepo.CreateReturns(nil, errors.New("watermelon"))
			})

			It("returns a meaningful error", func() {
				_, err := sandboxRepo.Create("some-sandbox-name")
				Expect(err).To(MatchError("create namespace: watermelon"))
			})
		})

		Context("if the sandbox already exists", func() {
			BeforeEach(func() {
				_, err := sandboxRepo.Create("some-sandbox-name")
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error", func() {
				_, err := sandboxRepo.Create("some-sandbox-name")
				Expect(err).To(MatchError(`sandbox "some-sandbox-name" already exists`))
			})

			It("locks and unlocks", func() {
				sandboxRepo.Create("some-sandbox-name")

				Expect(locker.LockCallCount()).To(Equal(2))
				Expect(locker.UnlockCallCount()).To(Equal(2))
			})
		})
	})

	Describe("Get", func() {
		It("locks and unlocks", func() {
			sandboxRepo.Get("some-other-sandbox-name")
			Expect(locker.LockCallCount()).To(Equal(1))
			Expect(locker.UnlockCallCount()).To(Equal(1))
		})

		Context("when getting a sandbox that hasn't been created", func() {
			It("returns an error", func() {
				sbox, err := sandboxRepo.Get("unknown-sandbox")
				Expect(sbox).To(BeNil())
				Expect(err).To(BeIdenticalTo(sandbox.NotFoundError))
			})
		})

		Context("when getting a sandbox that has been created", func() {
			It("returns the sandbox", func() {
				expectedSandbox, err := sandboxRepo.Create("some-sandbox-name")
				Expect(err).NotTo(HaveOccurred())

				sbox, err := sandboxRepo.Get("some-sandbox-name")
				Expect(err).NotTo(HaveOccurred())
				Expect(sbox).To(Equal(expectedSandbox))
			})
		})
	})

	Describe("Remove", func() {
		var otherSandbox sandbox.Sandbox

		BeforeEach(func() {
			_, err := sandboxRepo.Create("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())

			otherSandbox, err = sandboxRepo.Create("some-other-sandbox-name")
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes the sandbox by name", func() {
			sandboxRepo.Remove("some-sandbox-name")

			sbox, err := sandboxRepo.Get("some-sandbox-name")
			Expect(err).To(BeIdenticalTo(sandbox.NotFoundError))
			Expect(sbox).To(BeNil())
		})

		It("locks and unlocks", func() {
			sandboxRepo.Remove("some-other-sandbox-name")
			Expect(locker.LockCallCount()).To(Equal(3))
			Expect(locker.UnlockCallCount()).To(Equal(3))
		})

		It("does not remove other sandbox", func() {
			sandboxRepo.Remove("some-sandbox-name")

			sbox, err := sandboxRepo.Get("some-other-sandbox-name")
			Expect(err).NotTo(HaveOccurred())
			Expect(sbox).To(Equal(otherSandbox))
		})
	})
})
