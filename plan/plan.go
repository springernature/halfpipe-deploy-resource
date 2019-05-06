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

func (p Plan) Execute(executor Executor, logger logger.CapturingWriter) (err error) {
	for _, command := range p {
		logger.Println(fmt.Sprintf("$ %s", command))
		_, err = executor.CliCommand(command.Args()...)
		if err != nil {
			return
		}
		logger.Println()
	}
	return
}

func (p Plan) IsEmpty() bool {
	return len(p) == 0
}
