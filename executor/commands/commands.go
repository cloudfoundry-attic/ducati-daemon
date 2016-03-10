package commands

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

func All(commands ...executor.Command) executor.Command {
	return Group(commands)
}

type Group []executor.Command

func (g Group) Execute(context executor.Context) error {
	for i, c := range g {
		err := c.Execute(context)
		if err != nil {
			return &GroupError{
				index: i,
				group: g,
				Err:   err,
			}
		}
	}
	return nil
}

type GroupError struct {
	index int
	group Group
	Err   error
}

func (ge *GroupError) Error() string {
	return fmt.Sprintf("%s: commands: %s", ge.Err.Error(), toString(ge.group, ge.index))
}

func (g Group) String() string {
	return toString(g, -1)
}

func toString(group Group, cursor int) string {
	var buffer bytes.Buffer

	buffer.WriteString("(\n")
	for i, command := range group {
		cmdStr := command.String()
		if _, isGroup := command.(Group); isGroup {
			cmdStr = strings.Replace(cmdStr, "\n", "\n    ", -1)
		}

		if i == cursor {
			buffer.WriteString(fmt.Sprintf("--> %s", cmdStr))
		} else {
			buffer.WriteString(fmt.Sprintf("    %s", cmdStr))
		}

		if i < len(group)-1 {
			buffer.WriteString(" &&")
		}

		buffer.WriteString("\n")
	}
	buffer.WriteString(")")

	return buffer.String()
}
