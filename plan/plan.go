package plan

import (
	"errors"
	"fmt"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/gookit/color"
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"strings"
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
		logger.Println(color.New(color.FgGreen).Sprintf("$ %s", c.String()))

		errChan := make(chan error, 1)

		go func() {
			switch cmd := c.(type) {
			case clientCommand:
				errChan <- cmd.CallWithCfClient(cfClient, logger)

			case compoundCommand:
				_, err = executor.CliCommand(cmd.left)
				if cmd.shouldExecute(logger.BytesWritten) {
					logger.Println("")
					logger.Println("Failed to push/start application")

					// Here we know that we have failed for either
					// a. cf push failure
					// b. insufficient resources (which means all instances cannot be started due to lacking resources in CF)

					// If we have a err, we can assume its the `a` option that have failed
					if err != nil {
						logger.Println(fmt.Sprintf("$ %s", cmd.right))
						executor.CliCommand(cmd.right)
						errChan <- err
					} else {
						// This is due to insufficient resources. Maybe?
						if strings.Contains(string(logger.BytesWritten), `insufficient resources: memory`) {
							logger.Println(`insufficient resources means that CF is not scaled properly, please contact us in #ee`)
							errChan <- errors.New("failed to push/start application")
						}
					}
				}
				errChan <- err
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
