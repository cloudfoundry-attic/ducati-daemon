package sandbox_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path"

	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	"github.com/cloudfoundry-incubator/ducati-daemon/sandbox"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Sandbox Repository", func() {
	var (
		logger          *lagertest.TestLogger
		locker          *fakes.Locker
		sboxNamespace   *fakes.Namespace
		namespaceRepo   *fakes.Repository
		invoker         *fakes.Invoker
		sandboxRepo     *sandbox.Repository
		linkFactory     *fakes.LinkFactory
		missWatcher     *fakes.MissWatcher
		sandboxCallback *fakes.SandboxCallback
		sandboxFactory  *fakes.SandboxFactory
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		locker = &fakes.Locker{}
		invoker = &fakes.Invoker{}
		sboxNamespace = &fakes.Namespace{}
		namespaceRepo = &fakes.Repository{}
		namespaceRepo.CreateReturns(sboxNamespace, nil)
		linkFactory = &fakes.LinkFactory{}
		sandboxCallback = &fakes.SandboxCallback{}
		missWatcher = &fakes.MissWatcher{}

		sandboxFactory = &fakes.SandboxFactory{}
		sandboxFactory.NewStub = sandbox.New

		sandboxRepo = &sandbox.Repository{
			Logger:         logger,
			Locker:         locker,
			NamespaceRepo:  namespaceRepo,
			Invoker:        invoker,
			LinkFactory:    linkFactory,
			Watcher:        missWatcher,
			SandboxFactory: sandboxFactory,
			Sandboxes:      map[string]sandbox.Sandbox{},
		}
	})

	Describe("ForEach", func() {
		BeforeEach(func() {
			sbox, err := sandboxRepo.Create("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())
			Expect(sbox).NotTo(BeNil())

			sandboxCallback.CallbackReturns(nil)
			sboxNamespace.NameReturns("some-sandbox-name")
		})

		It("executes the callback for each sandbox in the repo", func() {
			err := sandboxRepo.ForEach(sandboxCallback)
			Expect(err).NotTo(HaveOccurred())

			Expect(sandboxCallback.CallbackCallCount()).To(Equal(1))
			ns := sandboxCallback.CallbackArgsForCall(0)
			Expect(ns).To(Equal(sboxNamespace))
		})

		It("locks and unlocks", func() {
			Expect(locker.LockCallCount()).To(Equal(1))
			Expect(locker.UnlockCallCount()).To(Equal(1))

			err := sandboxRepo.ForEach(sandboxCallback)
			Expect(err).NotTo(HaveOccurred())

			Expect(locker.LockCallCount()).To(Equal(2))
			Expect(locker.UnlockCallCount()).To(Equal(2))
		})

		Context("when the callback fails", func() {
			It("returns an error", func() {
				sandboxCallback.CallbackReturns(errors.New("potato"))

				err := sandboxRepo.ForEach(sandboxCallback)
				Expect(err).To(MatchError("callback: potato"))
			})
		})
	})

	Describe("Load", func() {
		var (
			sboxFile         *os.File
			sboxFileName     string
			sboxNamespaceDir string
		)

		BeforeEach(func() {
			var err error
			sboxNamespaceDir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())
			sboxFile, err = ioutil.TempFile(sboxNamespaceDir, "test")
			sboxFileName = path.Base(sboxFile.Name())
			Expect(err).NotTo(HaveOccurred())

			namespaceRepo.GetReturns(sboxNamespace, nil)
		})

		It("reads in files from the sanboxNamespaceDir into memory", func() {
			sbox, err := sandboxRepo.Get(sboxFileName)
			Expect(err).To(Equal(sandbox.NotFoundError))
			Expect(sbox).To(BeNil())

			err = sandboxRepo.Load(sboxNamespaceDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(namespaceRepo.GetCallCount()).To(Equal(1))
			Expect(namespaceRepo.GetArgsForCall(0)).To(Equal(sboxFileName))

			sbox, err = sandboxRepo.Get(sboxFileName)
			Expect(err).NotTo(HaveOccurred())
			Expect(sbox).NotTo(BeNil())
		})

		It("locks and unlocks", func() {
			err := sandboxRepo.Load(sboxNamespaceDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(locker.LockCallCount()).To(Equal(1))
			Expect(locker.UnlockCallCount()).To(Equal(1))
		})

		Context("when namespace repo fails to get sandbox", func() {
			It("returns an error", func() {
				namespaceRepo.GetReturns(nil, errors.New("potato"))

				err := sandboxRepo.Load(sboxNamespaceDir)
				Expect(err).To(MatchError(ContainSubstring("loading sandbox repo: potato")))
			})
		})
	})

	Describe("Create", func() {
		var fakeSandbox *fakes.Sandbox

		BeforeEach(func() {
			fakeSandbox = &fakes.Sandbox{}
			sandboxFactory.NewReturns(fakeSandbox)
		})

		It("returns the created sandbox", func() {
			sbox, err := sandboxRepo.Create("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())
			Expect(sbox).NotTo(BeNil())
		})

		It("logs entry and exit", func() {
			_, err := sandboxRepo.Create("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(logger).To(gbytes.Say("create.starting.*some-sandbox-name"))
			Expect(logger).To(gbytes.Say("create.complete.*some-sandbox-name"))
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

		It("injects the correct dependencies to the sandbox", func() {
			_, err := sandboxRepo.Create("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(sandboxFactory.NewCallCount()).To(Equal(1))
			log, ns, i, lf, w := sandboxFactory.NewArgsForCall(0)
			Expect(log).To(Equal(logger))
			Expect(ns).To(Equal(sboxNamespace))
			Expect(i).To(Equal(invoker))
			Expect(lf).To(Equal(linkFactory))
			Expect(w).To(Equal(missWatcher))
		})

		It("drives setup on the sandbox", func() {
			_, err := sandboxRepo.Create("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeSandbox.SetupCallCount()).To(Equal(1))
		})

		Context("if the sandbox already exists", func() {
			BeforeEach(func() {
				_, err := sandboxRepo.Create("some-sandbox-name")
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error", func() {
				_, err := sandboxRepo.Create("some-sandbox-name")
				Expect(err).To(Equal(sandbox.AlreadyExistsError))
			})

			It("locks and unlocks", func() {
				sandboxRepo.Create("some-sandbox-name")

				Expect(locker.LockCallCount()).To(Equal(2))
				Expect(locker.UnlockCallCount()).To(Equal(2))
			})
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

		Context("when setup fails", func() {
			BeforeEach(func() {
				fakeSandbox.SetupReturns(errors.New("dingleberry"))
			})

			It("returns a meaningful error", func() {
				_, err := sandboxRepo.Create("some-sandbox-name")
				Expect(err).To(MatchError("setup sandbox: dingleberry"))
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

	Describe("Destroy", func() {
		var sbox, otherSbox *fakes.Sandbox

		BeforeEach(func() {
			sbox = &fakes.Sandbox{}
			sbox.NamespaceReturns(sboxNamespace)
			sandboxRepo.Sandboxes["some-sandbox-name"] = sbox

			otherSbox = &fakes.Sandbox{}
			sandboxRepo.Sandboxes["some-other-sandbox-name"] = otherSbox
		})

		It("locks and unlocks", func() {
			err := sandboxRepo.Destroy("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(locker.LockCallCount()).To(Equal(1))
			Expect(locker.UnlockCallCount()).To(Equal(1))
		})

		It("logs entry and exit", func() {
			err := sandboxRepo.Destroy("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(logger).To(gbytes.Say("destroy.starting.*some-sandbox-name"))
			Expect(logger).To(gbytes.Say("destroy.complete.*some-sandbox-name"))
		})

		It("tears down the sandbox", func() {
			err := sandboxRepo.Destroy("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(sbox.TeardownCallCount()).To(Equal(1))
		})

		It("removes the sandbox by name", func() {
			err := sandboxRepo.Destroy("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(sandboxRepo.Sandboxes).NotTo(HaveKey("some-sandbox-name"))
		})

		It("removes the sandbox namespace", func() {
			err := sandboxRepo.Destroy("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(sbox.NamespaceCallCount()).To(Equal(1))
			Expect(namespaceRepo.DestroyCallCount()).To(Equal(1))

			ns := namespaceRepo.DestroyArgsForCall(0)
			Expect(ns).To(Equal(sboxNamespace))
		})

		It("does not remove other sandbox", func() {
			err := sandboxRepo.Destroy("some-sandbox-name")
			Expect(err).NotTo(HaveOccurred())

			sb, err := sandboxRepo.Get("some-other-sandbox-name")
			Expect(err).NotTo(HaveOccurred())
			Expect(sb).To(Equal(otherSbox))
		})

		Context("when the sandbox does not exist", func() {
			It("returns a NotFoundError", func() {
				err := sandboxRepo.Destroy("some-non-existent-name")
				Expect(err).To(BeIdenticalTo(sandbox.NotFoundError))
			})
		})

		Context("when teardown fails", func() {
			BeforeEach(func() {
				sbox.TeardownReturns(errors.New("papaya"))
			})

			It("returns a meaningful error", func() {
				err := sandboxRepo.Destroy("some-sandbox-name")
				Expect(err).To(MatchError("teardown: papaya"))
			})

			It("does not remove the sandbox entry", func() {
				sandboxRepo.Destroy("some-sandbox-name")
				Expect(sbox.TeardownCallCount()).To(Equal(1))

				Expect(sandboxRepo.Sandboxes).To(HaveKey("some-sandbox-name"))
			})

			It("does not destroy the namespace", func() {
				sandboxRepo.Destroy("some-sandbox-name")
				Expect(sbox.TeardownCallCount()).To(Equal(1))
				Expect(namespaceRepo.DestroyCallCount()).To(Equal(0))
			})
		})

		Context("when destroying the namespace fails", func() {
			BeforeEach(func() {
				namespaceRepo.DestroyReturns(errors.New("clementine"))
			})

			It("returns a meaningful error", func() {
				err := sandboxRepo.Destroy("some-sandbox-name")
				Expect(err).To(MatchError("namespace destroy: clementine"))
			})

			It("does not remove the sandbox entry", func() {
				sandboxRepo.Destroy("some-sandbox-name")
				Expect(namespaceRepo.DestroyCallCount()).To(Equal(1))

				Expect(sandboxRepo.Sandboxes).To(HaveKey("some-sandbox-name"))
			})
		})
	})
})
