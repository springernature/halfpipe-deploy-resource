package plan

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDoesTheNeedful(t *testing.T) {
	man := manifestparser.Application{
		Name: "myApp",
	}

	expectedPlan := Plan{
		NewCfCommand("delete", "myApp-CANDIDATE", "-f"),
	}

	summary := []cfclient.AppSummary{
		{
			Name:  "myApp",
			State: "started",
		},
		{
			Name:  "myApp-CANDIDATE",
			State: "started",
		},
	}

	p := NewDeleteCandidatePlan().Plan(man, summary)

	assert.Equal(t, expectedPlan, p)

}

func TestDoesNothingWhenThereIsNoCandidate(t *testing.T) {
	man := manifestparser.Application{
		Name: "myApp",
	}

	summary := []cfclient.AppSummary{
		{
			Name:  "myApp",
			State: "started",
		},
	}

	p := NewDeleteCandidatePlan().Plan(man, summary)

	assert.Equal(t, Plan{}, p)

}
