package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("unless", func() {
	var (
		context   *fakes.Context
		condition *fakes.Condition
		command   *fakes.Command
		unless    commands.Unless
	)

	BeforeEach(func() {
		context = &fakes.Context{}

		condition = &fakes.Condition{}
		condition.StringReturns("condition")

		command = &fakes.Command{}
		command.StringReturns("command")

		unless = commands.Unless{
			Condition: condition,
			Command:   command,
		}
	})

	Describe("String", func() {
		It("returns a string representation of the command", func() {
			Expect(unless.String()).To(Equal("condition || command"))
		})
	})

	Context("when the condition is satisfied", func() {
		BeforeEach(func() {
			condition.SatisfiedReturns(true, nil)
		})

		It("does not execute the command", func() {
			err := unless.Execute(context)
			Expect(err).NotTo(HaveOccurred())
			Expect(command.ExecuteCallCount()).To(Equal(0))
		})
	})

	Context("when the condition is not satisfied", func() {
		BeforeEach(func() {
			condition.SatisfiedReturns(false, nil)
		})

		It("executes the command", func() {
			err := unless.Execute(context)
			Expect(err).NotTo(HaveOccurred())
			Expect(command.ExecuteCallCount()).To(Equal(1))
			Expect(command.ExecuteArgsForCall(0)).To(Equal(context))
		})

		Context("when the command returns an error", func() {
			BeforeEach(func() {
				command.ExecuteReturns(errors.New("go away"))
			})

			It("wraps and propagates the error", func() {
				err := unless.Execute(context)
				Expect(err).To(MatchError("unless: go away"))
			})
		})
	})

	Context("when the condition check errors", func() {
		BeforeEach(func() {
			condition.SatisfiedReturns(false, errors.New("potato"))
		})
		It("returns an error and does not execute the command", func() {

			err := unless.Execute(context)
			Expect(err).To(MatchError("condition check: potato"))
			Expect(command.ExecuteCallCount()).To(Equal(0))
		})

	})
})
