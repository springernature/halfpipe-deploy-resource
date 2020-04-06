package plan

import (
	"fmt"
	"github.com/springernature/halfpipe-deploy-resource/logger"
)

type Plan []Command

func (p Plan) String() (s string) {
	if p.IsEmpty() {
		s += "# Nothing to do!"
		return
	}

	s += "# Planned execution\n"
	for _, command := range p {
		s += fmt.Sprintf("#\t* %s\n", command)
	}
	return
}

func (p Plan) Execute(executor Executor, logger *logger.CapturingWriter) (err error) {
	for _, c := range p {
		logger.Println(fmt.Sprintf("$ %s", c))

		var command Command
		var onFailure Command
		var shouldExecuteOnFailure ShouldExecute

		switch cmd := c.(type) {
		case compoundCommand:
			command = cmd.left
			onFailure = cmd.right
			shouldExecuteOnFailure = cmd.shouldExecute
		case Command:
			command = cmd
		}

		_, err = executor.CliCommand(command)
		if err != nil {
			if shouldExecuteOnFailure != nil && shouldExecuteOnFailure(logger.BytesWritten) {
				logger.Println("")
				logger.Println("Failed to push/start application")
				logger.Println(fmt.Sprintf("$ %s", onFailure))
				executor.CliCommand(onFailure)
			}
			return
		}
		logger.Println()
	}
	return
}

func (p Plan) IsEmpty() bool {
	return len(p) == 0
}
