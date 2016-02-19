package commands_test

import (
	"errors"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ExecuteInNamespace", func() {
	var (
		context   *fakes.Context
		namespace *fakes.Namespace
		command   *fakes.Command

		inNamespace commands.InNamespace
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		command = &fakes.Command{}

		namespace = &fakes.Namespace{}
		namespace.ExecuteStub = func(callback func(*os.File) error) error {
			Expect(command.ExecuteCallCount()).To(Equal(0))

			err := callback(nil)

			Expect(command.ExecuteCallCount()).To(Equal(1))
			return err
		}

		inNamespace = commands.InNamespace{
			Namespace: namespace,
			Command:   command,
		}
	})

	It("executes the command in the specified namespace", func() {
		err := inNamespace.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(namespace.ExecuteCallCount()).To(Equal(1))
	})

	It("executes the command with the correct context", func() {
		err := inNamespace.Execute(context)
		Expect(err).NotTo(HaveOccurred())

		Expect(command.ExecuteCallCount()).To(Equal(1))
		Expect(command.ExecuteArgsForCall(0)).To(Equal(context))
	})

	Context("when namespace execute fails", func() {
		BeforeEach(func() {
			namespace.ExecuteReturns(errors.New("go away"))
		})

		It("propagates the error", func() {
			err := inNamespace.Execute(context)
			Expect(err).To(MatchError("go away"))
		})
	})

	Context("when the target command fails", func() {
		BeforeEach(func() {
			command.ExecuteReturns(errors.New("i died"))
		})

		It("propagates the error", func() {
			err := inNamespace.Execute(context)
			Expect(err).To(MatchError("i died"))
		})
	})
})
