package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("If", func() {
	var (
		context   *fakes.Context
		condition *fakes.Condition
		command   *fakes.Command
		ifCommand commands.If
	)

	BeforeEach(func() {
		context = &fakes.Context{}
		condition = &fakes.Condition{}
		command = &fakes.Command{}

		ifCommand = commands.If{
			Condition: condition,
			Command:   command,
		}
	})

	Context("when the condition is satisfied", func() {
		BeforeEach(func() {
			condition.SatisfiedReturns(true)
		})

		It("executes the command with the provided context", func() {
			err := ifCommand.Execute(context)
			Expect(err).NotTo(HaveOccurred())
			Expect(command.ExecuteCallCount()).To(Equal(1))
			Expect(command.ExecuteArgsForCall(0)).To(Equal(context))
		})

		Context("when the command returns an error", func() {
			BeforeEach(func() {
				command.ExecuteReturns(errors.New("go away"))
			})

			It("propagates the error", func() {
				err := ifCommand.Execute(context)
				Expect(err).To(MatchError("go away"))
			})
		})
	})

	Context("when the condition is not satisfied", func() {
		BeforeEach(func() {
			condition.SatisfiedReturns(false)
		})

		It("does not execute the commands", func() {
			err := ifCommand.Execute(context)
			Expect(err).NotTo(HaveOccurred())
			Expect(command.ExecuteCallCount()).To(Equal(0))
		})
	})
})
