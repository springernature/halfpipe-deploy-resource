package plan

import (
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"testing"

	"errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
)

var discardLogger = logger.NewLogger(ioutil.Discard)

type mockExecutor struct {
	err error
	f   func(command Command) ([]string, error)
}

func newMockExecutorWithError(err error) Executor {
	return mockExecutor{
		err: err,
	}
}

func newMockExecutorWithFunction(fun func(command Command) ([]string, error)) Executor {
	return mockExecutor{
		f: fun,
	}
}

func (m mockExecutor) CliCommand(command Command) ([]string, error) {
	if m.f != nil {
		return m.f(command)
	}
	return []string{}, m.err
}

func TestPlan_String(t *testing.T) {
	p := Plan{
		NewCfCommand("push"),
		NewCfCommand("delete"),
	}

	expected := `# Planned execution
#	* cf push
#	* cf delete
`
	assert.Equal(t, expected, p.String())
}

func TestPlan_ExecutePassesOnError(t *testing.T) {
	expectedError := errors.New("expected error")

	p := Plan{
		NewCfCommand("error"),
	}

	err := p.Execute(newMockExecutorWithError(expectedError), &cfclient.Client{}, &discardLogger)

	assert.Equal(t, expectedError, err)
}

func TestPlan_ExecutePassesOnErrorIfItHappensInTheMiddleOfThePlan(t *testing.T) {
	expectedError := errors.New("expected error")
	var numberOfCalls int

	p := Plan{
		NewCfCommand("ok"),
		NewCfCommand("ok"),
		NewCfCommand("error"),
		NewCfCommand("ok"),
	}

	err := p.Execute(newMockExecutorWithFunction(func(command Command) ([]string, error) {
		numberOfCalls++
		if command.Args()[0] == "error" {
			return []string{}, expectedError
		}
		return []string{}, nil
	}),&cfclient.Client{}, &discardLogger)

	assert.Equal(t, 3, numberOfCalls)
	assert.Equal(t, expectedError, err)
}

func TestPlan_Execute(t *testing.T) {
	var numberOfCalls int

	p := Plan{
		NewCfCommand("ok"),
		NewCfCommand("ok"),
		NewCfCommand("ok"),
		NewCfCommand("ok"),
	}

	err := p.Execute(newMockExecutorWithFunction(func(command Command) ([]string, error) {
		numberOfCalls++
		return []string{}, nil
	}), &cfclient.Client{}, &discardLogger)

	assert.Nil(t, err)
	assert.Equal(t, 4, numberOfCalls)
}
