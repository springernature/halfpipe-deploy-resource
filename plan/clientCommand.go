package plan

import (
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/logger"
)

type ClientCommand interface {
	CallWithCfClient(client *cfclient.Client, logger *logger.CapturingWriter) error
}

type clientCommand struct {
	fun func(client *cfclient.Client, logger *logger.CapturingWriter) error
}

func (c clientCommand) CallWithCfClient(client *cfclient.Client, logger *logger.CapturingWriter) error {
	return c.fun(client, logger)
}

func (c clientCommand) String() string {
	// Todo this should be configurable
	return "Checking that all app instances are in running state"
}

func (c clientCommand) Args() []string {
	panic("Args should never be called on a clientCommand")
}

func (c clientCommand) Env() []string {
	panic("Env should never be called on a clientCommand")
}

func (c clientCommand) AddToArgs(args ...string) Command {
	panic("AddToArgs should never be called on a clientCommand")
}

func (c clientCommand) AddToEnv(env ...string) Command {
	panic("AddToEnv should never be called on a clientCommand")
}

func (c clientCommand) Cmd() string {
	return ""
}

func NewClientCommand(fun func(client *cfclient.Client, logger *logger.CapturingWriter) error) Command {
	return clientCommand{
		fun: fun,
	}
}
