package plan

import (
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"os/exec"
)

type Executor interface {
	CliCommand(args ...string) ([]string, error)
}

type cfCLIExecutor struct {
	logger *logger.CapturingWriter
}

// This executor differs from the executor used in the plugin in that it
// executes CF binary through the operating system rather than through the plugin system.
func NewCFCliExecutor(logger *logger.CapturingWriter) Executor {
	return cfCLIExecutor{
		logger: logger,
	}
}

func (c cfCLIExecutor) CliCommand(args ...string) (out []string, err error) {
	execCmd := exec.Command("cf", args...) // #nosec disables the gas warning for this line.
	execCmd.Stdout = c.logger
	execCmd.Stderr = c.logger

	if err = execCmd.Start(); err != nil {
		return
	}

	if err = execCmd.Wait(); err != nil {
		return
	}

	return
}
