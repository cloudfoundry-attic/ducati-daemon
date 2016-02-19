package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/commands/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Commands", func() {
	Describe("All", func() {
		var (
			context  *fakes.Context
			command1 *fakes.Command
			command2 *fakes.Command
			command3 *fakes.Command
			all      commands.Command
		)

		BeforeEach(func() {
			context = &fakes.Context{}

			command1 = &fakes.Command{}
			command2 = &fakes.Command{}
			command3 = &fakes.Command{}

			all = commands.All(command1, command2, command3)
		})

		It("creates a command wrapper that executes all commands", func() {
			err := all.Execute(context)
			Expect(err).NotTo(HaveOccurred())

			Expect(command1.ExecuteCallCount()).To(Equal(1))
			Expect(command1.ExecuteArgsForCall(0)).To(Equal(context))

			Expect(command2.ExecuteCallCount()).To(Equal(1))
			Expect(command2.ExecuteArgsForCall(0)).To(Equal(context))

			Expect(command3.ExecuteCallCount()).To(Equal(1))
			Expect(command3.ExecuteArgsForCall(0)).To(Equal(context))
		})

		Context("when a command returns an error", func() {
			BeforeEach(func() {
				command2.ExecuteReturns(errors.New("go away"))
			})

			It("propagates the error", func() {
				err := all.Execute(context)
				Expect(err).To(MatchError("go away"))
			})

			It("stops execution after first failure", func() {
				all.Execute(context)
				Expect(command1.ExecuteCallCount()).To(Equal(1))
				Expect(command2.ExecuteCallCount()).To(Equal(1))
				Expect(command3.ExecuteCallCount()).To(Equal(0))
			})
		})
	})
})
