package plan

import (
	"errors"
	"fmt"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/fatih/color"
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"time"
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

func (p Plan) Execute(executor Executor, cfClient *cfclient.Client, logger *logger.CapturingWriter, timeout time.Duration) (err error) {
	for _, c := range p {
		color.New(color.FgGreen).Fprintln(logger.Writer, fmt.Sprintf("$ %s", c))

		errChan := make(chan error, 1)

		go func() {
			switch cmd := c.(type) {
			case clientCommand:
				errChan <- cmd.CallWithCfClient(cfClient, logger)

			case compoundCommand:
				_, err = executor.CliCommand(cmd.left)
				if err != nil {
					if cmd.shouldExecute(logger.BytesWritten) {
						logger.Println("")
						logger.Println("Failed to push/start application")
						logger.Println(fmt.Sprintf("$ %s", cmd.right))
						_, err = executor.CliCommand(cmd.right)
						errChan <- err
					} else {
						errChan <- err
					}
				}
			case Command:
				_, err = executor.CliCommand(cmd)
				errChan <- err
			}
		}()

		select {
		case err = <-errChan:
			if err != nil {
				return
			}
		case <-time.After(timeout):
			return errors.New(fmt.Sprintf("command time out after %s", timeout.String()))
		}
		logger.Println()
	}
	return
}

func (p Plan) IsEmpty() bool {
	return len(p) == 0
}
