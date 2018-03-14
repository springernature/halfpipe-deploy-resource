package plan

import (
	"fmt"
	"log"
	"strings"
	"github.com/fatih/color"
	"regexp"
	"code.cloudfoundry.org/cli/util/manifest"
)

type Command struct {
	command string
	args    []string
}

func NewCfCommand(args ...string) Command {
	return Command{
		command: "cf",
		args:    args,
	}
}

func (c Command) String() string {
	var commandArgs = strings.Join(c.args, " ")

	if strings.HasPrefix(commandArgs, "login") {
		// If the command is login, a dirty replace of whatever comes after "-p "
		// to hide cf password from concourse console output
		cfLoginPasswordRegex := regexp.MustCompile(`-p ([a-zA-Z0-9_-]+)`)
		commandArgs = cfLoginPasswordRegex.ReplaceAllLiteralString(commandArgs, "-p ********")
	}

	return fmt.Sprintf("%s %s", c.command, commandArgs)
}

type Plan []Command

func (c Plan) String() (s string) {
	s += "Planned execution\n"
	for _, p := range c {
		s += fmt.Sprintf("\t* %s\n", p)
	}
	return
}

func (c Plan) Execute(executor Executor, logger *log.Logger, col *color.Color) (err error) {
	for _, p := range c {
		logger.Println(col.Sprintf("=== Executing '%s' ===", p))
		_, err = executor.CliCommand(p.args...)
		if err != nil {
			return
		}
		logger.Println(col.Sprintf("=== Succeeded :D ==="))
		logger.Println()
	}
	return
}

type Planner interface {
	GetPlan(application manifest.Application) (Plan, error)
}

type Executor interface {
	CliCommand(args ...string) ([]string, error)
}