package plan

import (
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDoesNothingWhenNoAppsToCleanup(t *testing.T)  {
	summary := []cfclient.AppSummary{
		{
			Name:  "myApp",
			State: "started",
		},
	}

	man := manifest.Application{
		Name: "myApp",
	}

	p := NewCleanupPlan().Plan(man, summary)
	assert.Empty(t, p)

}

func TestDeletesAppThatNeedsACleanup(t *testing.T)  {
	summary := []cfclient.AppSummary{
		{
			Name:  "myApp",
			State: "started",
		},	{
			Name:  "myApp-DELETE",
			State: "stopped",
		},
	}

	man := manifest.Application{
		Name: "myApp",
	}

	expectedPlan := Plan{
		NewCfCommand("delete", "myApp-DELETE"),
	}

	p := NewCleanupPlan().Plan(man, summary)

	assert.Equal(t, expectedPlan, p)
}


func TestDeletesAppsThatNeedsACleanup(t *testing.T)  {
	summary := []cfclient.AppSummary{
		{
			Name:  "myApp",
			State: "started",
		},
		{
			Name:  "myApp-DELETE",
			State: "stopped",
		},{
			Name:  "myApp-DELETE-1",
			State: "stopped",
		}, {
			Name:  "myApp-DELETE-2",
			State: "stopped",
		},{
			Name:  "somethingElse-DELETE-2",
			State: "stopped",
		},
	}

	man := manifest.Application{
		Name: "myApp",
	}

	expectedPlan := Plan{
		NewCfCommand("delete", "myApp-DELETE"),
		NewCfCommand("delete", "myApp-DELETE-1"),
		NewCfCommand("delete", "myApp-DELETE-2"),
	}

	p := NewCleanupPlan().Plan(man, summary)

	assert.Equal(t, expectedPlan, p)
}
