package plan

import (
	"fmt"
	"regexp"
	"strings"
)

type Command interface {
	fmt.Stringer
	Args() []string
	AddToArgs(args ...string) Command
}

type cfCommand struct {
	command string
	args    []string
}

func NewCfCommand(args ...string) Command {
	return cfCommand{
		command: "cf",
		args:    args,
	}
}

func (c cfCommand) Args() []string {
	return c.args
}

func (c cfCommand) AddToArgs(args ...string) Command {
	c.args = append(c.args, args...)
	return c
}

func (c cfCommand) String() string {
	var commandArgs = strings.Join(c.args, " ")

	if strings.HasPrefix(commandArgs, "login") {
		// If the command is login, a dirty replace of whatever comes after "-p "
		// to hide cf password from concourse console output
		cfLoginPasswordRegex := regexp.MustCompile(`-p ([a-zA-Z0-9_-]+)`)
		commandArgs = cfLoginPasswordRegex.ReplaceAllLiteralString(commandArgs, "-p ********")
	}

	return fmt.Sprintf("%s %s", c.command, commandArgs)
}
