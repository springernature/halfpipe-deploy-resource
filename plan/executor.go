package plan

import (
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"os"
	"os/exec"
)

type Executor interface {
	CliCommand(command Command) ([]string, error)
}

type cfCLIExecutor struct {
	logger    *logger.CapturingWriter
	cfVersion string
}

// This executor differs from the executor used in the plugin in that it
// executes CF binary through the operating system rather than through the plugin system.
func NewCFCliExecutor(logger *logger.CapturingWriter, request Request) Executor {
	return cfCLIExecutor{
		logger:    logger,
		cfVersion: request.Params.CliVersion,
	}
}

func (c cfCLIExecutor) CliCommand(command Command) (out []string, err error) {
	execCmd := exec.Command(c.cfVersion, command.Args()...) // #nosec disables the gas warning for this line.
	execCmd.Stdout = c.logger
	execCmd.Stderr = c.logger
	execCmd.Env = append(os.Environ(), command.Env()...)

	if err = execCmd.Start(); err != nil {
		return
	}

	if err = execCmd.Wait(); err != nil {
		return
	}

	return
}
