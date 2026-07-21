package plan

import (
	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"github.com/springernature/halfpipe-deploy-resource"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDoesTheNeedful(t *testing.T) {
	manifest := `applications:
- name: myApp
`
	man := halfpipe_deploy_resource.ParseManifest(manifest)

	expectedPlan := Plan{
		NewCfCommand("delete", "myApp-CANDIDATE", "-f"),
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

	p := NewDeleteCandidatePlan().Plan(man.Applications[0], summary)

	assert.Equal(t, expectedPlan, p)

}

func TestDoesNothingWhenThereIsNoCandidate(t *testing.T) {
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

	p := NewDeleteCandidatePlan().Plan(man.Applications[0], summary)

	assert.Equal(t, Plan{}, p)

}
