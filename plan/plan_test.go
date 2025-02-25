package plan

import (
	"fmt"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"testing"
	"time"

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

	err := p.Execute(newMockExecutorWithError(expectedError), &cfclient.Client{}, &discardLogger, 1*time.Second, false)

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
	}), &cfclient.Client{}, &discardLogger, 1*time.Minute, false)

	assert.Equal(t, 3, numberOfCalls)
	assert.Equal(t, expectedError, err)
}

func TestPlan_ExecuteErrorsWhenACommandTimesOut(t *testing.T) {
	expectedError := errors.New("command time out after 5ms")

	p := Plan{
		NewCfCommand("timeout"),
	}

	err := p.Execute(newMockExecutorWithFunction(func(command Command) ([]string, error) {
		time.Sleep(10 * time.Millisecond)
		return []string{}, nil
	}), &cfclient.Client{}, &discardLogger, 5*time.Millisecond, false)

	assert.Equal(t, expectedError, err)
}

func TestPlan_ExecuteErrorsWhenACompoundCommandTimesOut(t *testing.T) {
	expectedError := errors.New("command time out after 5ms")

	p := Plan{
		NewCompoundCommand(NewCfCommand("timeout"), NewCfCommand("blah"), func(log []byte) bool {
			return false
		}, false),
	}

	err := p.Execute(newMockExecutorWithFunction(func(command Command) ([]string, error) {
		time.Sleep(10 * time.Millisecond)
		return []string{}, nil
	}), &cfclient.Client{}, &discardLogger, 5*time.Millisecond, false)

	assert.Equal(t, expectedError, err)
}

func TestPlan_ExecuteErrorsWhenACommandWithClientTimesOut(t *testing.T) {
	expectedError := errors.New("command time out after 5ms")

	p := Plan{
		NewClientCommand(func(client *cfclient.Client, logger *logger.CapturingWriter) error {
			time.Sleep(6 * time.Millisecond)
			return nil
		}, "description"),
	}

	err := p.Execute(nil, &cfclient.Client{}, &discardLogger, 5*time.Millisecond, false)

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
	}), &cfclient.Client{}, &discardLogger, 1*time.Minute, false)

	assert.Nil(t, err)
	assert.Equal(t, 4, numberOfCalls)
}

func TestPlan_Compound(t *testing.T) {
	t.Run("Calls right, and then errors", func(t *testing.T) {
		expectedError := errors.New("meehp")
		var called []string
		p := Plan{
			NewCfCommand("1"),
			NewCompoundCommand(NewCfCommand("2"), NewCfCommand("3"), func(log []byte) bool {
				return true
			}, true),
			NewCfCommand("4"),
			NewCfCommand("5"),
		}

		err := p.Execute(newMockExecutorWithFunction(func(command Command) ([]string, error) {
			fmt.Println(command)
			called = append(called, command.Args()[0])
			if command.Args()[0] == "2" {
				return []string{}, expectedError
			}

			return []string{}, nil
		}), &cfclient.Client{}, &discardLogger, 1*time.Minute, false)

		assert.Equal(t, expectedError, err)
		assert.Equal(t, []string{"1", "2", "3"}, called)
	})

	t.Run("Calls right, and then continues", func(t *testing.T) {
		var called []string
		p := Plan{
			NewCfCommand("1"),
			NewCompoundCommand(NewCfCommand("2"), NewCfCommand("3"), func(log []byte) bool {
				return true
			}, false),
			NewCfCommand("4"),
			NewCfCommand("5"),
		}

		err := p.Execute(newMockExecutorWithFunction(func(command Command) ([]string, error) {
			called = append(called, command.Args()[0])
			if command.Args()[0] == "2" {
				return []string{}, errors.New("something to trigger the right")
			}
			return []string{}, nil
		}), &cfclient.Client{}, &discardLogger, 1*time.Minute, false)

		assert.NoError(t, err)
		assert.Equal(t, []string{"1", "2", "3", "4", "5"}, called)
	})

	t.Run("Calls right, but returns error in case it errors", func(t *testing.T) {
		expectedError := errors.New("meehp")
		var called []string
		p := Plan{
			NewCfCommand("1"),
			NewCompoundCommand(NewCfCommand("2"), NewCfCommand("3"), func(log []byte) bool {
				return true
			}, false),
			NewCfCommand("4"),
			NewCfCommand("5"),
		}

		err := p.Execute(newMockExecutorWithFunction(func(command Command) ([]string, error) {
			called = append(called, command.Args()[0])
			if command.Args()[0] == "2" {
				return []string{}, errors.New("something to trigger the right")
			}

			if command.Args()[0] == "3" {
				return []string{}, expectedError
			}
			return []string{}, nil
		}), &cfclient.Client{}, &discardLogger, 1*time.Minute, false)

		assert.Error(t, err)
		assert.Equal(t, []string{"1", "2", "3"}, called)
	})
}
