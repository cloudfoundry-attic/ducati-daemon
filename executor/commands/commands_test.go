package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor/commands/fakes"

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

			command1.StringReturns("some-command 1")
			command2.StringReturns("some-command 2")
			command3.StringReturns("some-command 3")

			all = commands.All(command1, command2, command3)
		})

		Describe("Execute", func() {
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
		})

		Context("when a command returns an error", func() {
			BeforeEach(func() {
				command2.ExecuteReturns(errors.New("go away"))
			})

			It("wraps the error with contextual information", func() {
				err := all.Execute(context)
				Expect(err.Error()).To(Equal("go away: commands: (\n" +
					"    some-command 1 &&\n" +
					"--> some-command 2 &&\n" +
					"    some-command 3\n" +
					")"))
			})

			It("makes the original error available", func() {
				err := all.Execute(context)
				Expect(err).To(HaveOccurred())

				groupError, ok := err.(*commands.GroupError)
				Expect(ok).To(BeTrue())
				Expect(groupError.Err).To(Equal(errors.New("go away")))
			})

			It("stops execution after first failure", func() {
				all.Execute(context)
				Expect(command1.ExecuteCallCount()).To(Equal(1))
				Expect(command2.ExecuteCallCount()).To(Equal(1))
				Expect(command3.ExecuteCallCount()).To(Equal(0))
			})
		})

		Describe("String", func() {
			var cmdStr string

			It("renders the list of commands in the group", func() {
				all.Execute(context)
				cmdStr = all.String()
				Expect(cmdStr).To(Equal("(\n" +
					"    some-command 1 &&\n" +
					"    some-command 2 &&\n" +
					"    some-command 3\n" +
					")"))
			})

			Context("when the command group is empty", func() {
				It("prints as two braces", func() {
					all = commands.Group([]commands.Command{})
					Expect(all.String()).To(Equal("(\n)"))
				})
			})

			Context("when there are nested commands", func() {
				It("indents the nested group with braces", func() {
					all = commands.All(commands.All(command1, command2), command3)
					all.Execute(context)
					cmdStr = all.String()
					Expect(cmdStr).To(Equal("(\n" +
						"    (\n" +
						"        some-command 1 &&\n" +
						"        some-command 2\n" +
						"    ) &&\n" +
						"    some-command 3\n" +
						")"))
				})
			})
		})
	})
})
