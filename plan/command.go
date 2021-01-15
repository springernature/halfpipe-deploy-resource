package plan

import (
	"fmt"
	"regexp"
	"strings"
)

type Command interface {
	fmt.Stringer
	Cmd() string
	Args() []string
	Env() []string
	AddToArgs(args ...string) Command
	AddToEnv(env ...string) Command
}

type command struct {
	command string
	args    []string
	env     []string
}

func (c command) Cmd() string {
	return c.command
}

func NewCfCommand(args ...string) Command {
	return command{
		command: "cf",
		args:    args,
	}
}

func (c command) Args() []string {
	return c.args
}

func (c command) Env() []string {
	return c.env
}

func (c command) AddToArgs(args ...string) Command {
	c.args = append(c.args, args...)
	return c
}

func (c command) AddToEnv(env ...string) Command {
	c.env = append(c.env, env...)
	return c
}

func (c command) String() string {
	var commandArgs = strings.Join(c.args, " ")

	if strings.HasPrefix(commandArgs, "login") {
		// If the command is login, a dirty replace of whatever comes after "-p "
		// to hide cf password from concourse console output
		cfLoginPasswordRegex := regexp.MustCompile(`-p ([a-zA-Z0-9_-]+)`)
		commandArgs = cfLoginPasswordRegex.ReplaceAllLiteralString(commandArgs, "-p ********")
	}
	cmd := fmt.Sprintf("%s %s", c.command, commandArgs)

	if len(c.env) == 0 {
		return cmd
	}

	var cleanedEnv []string
	for _, e := range c.env {
		key := strings.Split(e, "=")[0]
		cleanedEnv = append(cleanedEnv, fmt.Sprintf("%s=...", key))
	}
	env := strings.Join(cleanedEnv, " ")
	return fmt.Sprintf("%s %s", env, cmd)
}
