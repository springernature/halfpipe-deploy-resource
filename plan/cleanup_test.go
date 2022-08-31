package plan

import (
	"github.com/cloudfoundry-community/go-cfclient"
	halfpipe_deploy_resource "github.com/springernature/halfpipe-deploy-resource"
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

	man := halfpipe_deploy_resource.ParseManifest(`applications:
- name: MyApp`).Applications[0]

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

	man := halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp`).Applications[0]

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

	man := halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp`).Applications[0]

	expectedPlan := Plan{
		NewCfCommand("delete", "myApp-DELETE", "-f"),
		NewCfCommand("delete", "myApp-DELETE-1", "-f"),
		NewCfCommand("delete", "myApp-DELETE-2", "-f"),
	}

	p := NewCleanupPlan().Plan(man, summary)

	assert.Equal(t, expectedPlan, p)
}
