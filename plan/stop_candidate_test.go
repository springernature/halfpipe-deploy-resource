package plan

import (
	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"github.com/springernature/halfpipe-deploy-resource"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStopsCandidateWhenItExists(t *testing.T) {
	manifest := `applications:
- name: myApp
`
	man := halfpipe_deploy_resource.ParseManifest(manifest)

	expectedPlan := Plan{
		NewCfCommand("stop", "myApp-CANDIDATE"),
	}

	summary := []*resource.App{
		{
			Name:  "myApp",
			State: "started",
		},
		{
			Name:  "myApp-CANDIDATE",
			State: "started",
		},
	}

	p := NewStopCandidatePlan().Plan(man.Applications[0], summary)

	assert.Equal(t, expectedPlan, p)
}

func TestStopsStoppedCandidate(t *testing.T) {
	manifest := `applications:
- name: myApp
`
	man := halfpipe_deploy_resource.ParseManifest(manifest)

	expectedPlan := Plan{
		NewCfCommand("stop", "myApp-CANDIDATE"),
	}

	summary := []*resource.App{
		{
			Name:  "myApp",
			State: "started",
		},
		{
			Name:  "myApp-CANDIDATE",
			State: "stopped",
		},
	}

	p := NewStopCandidatePlan().Plan(man.Applications[0], summary)

	assert.Equal(t, expectedPlan, p)
}

func TestDoesNothingWhenThereIsNoCandidateToStop(t *testing.T) {
	manifest := `applications:
- name: myApp
`
	man := halfpipe_deploy_resource.ParseManifest(manifest)

	summary := []*resource.App{
		{
			Name:  "myApp",
			State: "started",
		},
	}

	p := NewStopCandidatePlan().Plan(man.Applications[0], summary)

	assert.Equal(t, Plan{}, p)
}
