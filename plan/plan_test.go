package plan

import (
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"testing"

	"errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
)

type mockExecutor struct {
	err error
	f   func(args ...string) ([]string, error)
}

func newMockExecutorWithError(err error) Executor {
	return mockExecutor{
		err: err,
	}
}

func newMockExecutorWithFunction(fun func(args ...string) ([]string, error)) Executor {
	return mockExecutor{
		f: fun,
	}
}

func (m mockExecutor) CliCommand(args ...string) ([]string, error) {
	if m.f != nil {
		return m.f(args...)
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

	err := p.Execute(newMockExecutorWithError(expectedError), logger.NewLogger(ioutil.Discard))

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

	err := p.Execute(newMockExecutorWithFunction(func(args ...string) ([]string, error) {
		numberOfCalls++
		if args[0] == "error" {
			return []string{}, expectedError
		}
		return []string{}, nil
	}), logger.NewLogger(ioutil.Discard))

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

	err := p.Execute(newMockExecutorWithFunction(func(args ...string) ([]string, error) {
		numberOfCalls++
		return []string{}, nil
	}), logger.NewLogger(ioutil.Discard))

	assert.Nil(t, err)
	assert.Equal(t, 4, numberOfCalls)
}
