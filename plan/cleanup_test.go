package plan

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDoesNothingWhenNoAppsToCleanup(t *testing.T) {
	summary := []cfclient.AppSummary{
		{
			Name:  "myApp",
			State: "started",
		},
	}

	man := manifestparser.Application{
		Name: "myApp",
	}

	p := NewCleanupPlan().Plan(man, summary)
	assert.Empty(t, p)

}

func TestDeletesAppThatNeedsACleanup(t *testing.T) {
	summary := []cfclient.AppSummary{
		{
			Name:  "myApp",
			State: "started",
		}, {
			Name:  "myApp-DELETE",
			State: "stopped",
		},
	}

	man := manifestparser.Application{
		Name: "myApp",
	}

	expectedPlan := Plan{
		NewCfCommand("delete", "myApp-DELETE", "-f"),
	}

	p := NewCleanupPlan().Plan(man, summary)

	assert.Equal(t, expectedPlan, p)
}

func TestDeletesAppsThatNeedsACleanup(t *testing.T) {
	summary := []cfclient.AppSummary{
		{
			Name:  "myApp",
			State: "started",
		},
		{
			Name:  "myApp-DELETE",
			State: "stopped",
		}, {
			Name:  "myApp-DELETE-1",
			State: "stopped",
		}, {
			Name:  "myApp-DELETE-2",
			State: "stopped",
		}, {
			Name:  "somethingElse-DELETE-2",
			State: "stopped",
		},
	}

	man := manifestparser.Application{
		Name: "myApp",
	}

	expectedPlan := Plan{
		NewCfCommand("delete", "myApp-DELETE", "-f"),
		NewCfCommand("delete", "myApp-DELETE-1", "-f"),
		NewCfCommand("delete", "myApp-DELETE-2", "-f"),
	}

	p := NewCleanupPlan().Plan(man, summary)

	assert.Equal(t, expectedPlan, p)
}
