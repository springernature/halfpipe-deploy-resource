package plan

import (
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWorkerApp(t *testing.T) {
	t.Run("No previously deployed version", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "started",
			},
		}

		man := manifest.Application{
			Name:    "myApp",
			NoRoute: true,
		}
		expectedPlan := Plan{
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan().Plan(man, validRequest, summary)
		//assert.Nil(t, err)
		assert.Equal(t, expectedPlan, plan)
	})

	t.Run("One previously deployed stopped version", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "started",
			},
			{
				Name:  "myApp",
				State: "stopped",
			},
		}

		man := manifest.Application{
			Name:    "myApp",
			NoRoute: true,
		}
		expectedPlan := Plan{
			NewCfCommand("rename", man.Name, createOldAppName(man.Name)),
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan().Plan(man, validRequest, summary)
		//assert.Nil(t, err)
		assert.Equal(t, expectedPlan, plan)
	})

	t.Run("One previously deployed started version", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "started",
			},
			{
				Name:  "myApp",
				State: "started",
			},
		}

		man := manifest.Application{
			Name:    "myApp",
			NoRoute: true,
		}
		expectedPlan := Plan{
			NewCfCommand("rename", man.Name, createOldAppName(man.Name)),
			NewCfCommand("stop", createOldAppName(man.Name)),
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan().Plan(man, validRequest, summary)
		assert.Equal(t, expectedPlan, plan)
	})

	t.Run("One previously deployed started version with an stopped old version", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "started",
			},
			{
				Name:  "myApp",
				State: "started",
			},
			{
				Name:  "myApp-OLD",
				State: "stopped",
			},
		}

		man := manifest.Application{
			Name:    "myApp",
			NoRoute: true,
		}
		expectedPlan := Plan{
			NewCfCommand("rename", createOldAppName(man.Name), createDeleteName(man.Name, 0)),
			NewCfCommand("rename", man.Name, createOldAppName(man.Name)),
			NewCfCommand("stop", createOldAppName(man.Name)),
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan().Plan(man, validRequest, summary)
		assert.Equal(t, expectedPlan, plan)
	})

	t.Run("One previously deployed started version with an stopped old version and a uncleaned up DELETE app", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "started",
			},
			{
				Name:  "myApp",
				State: "started",
			},
			{
				Name:  "myApp-OLD",
				State: "stopped",
			},
			{
				Name:  "myApp-DELETE",
				State: "stopped",
			},
		}

		man := manifest.Application{
			Name:    "myApp",
			NoRoute: true,
		}
		expectedPlan := Plan{
			NewCfCommand("rename", createOldAppName(man.Name), createDeleteName(man.Name, 1)),
			NewCfCommand("rename", man.Name, createOldAppName(man.Name)),
			NewCfCommand("stop", createOldAppName(man.Name)),
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan().Plan(man, validRequest, summary)
		assert.Equal(t, expectedPlan, plan)
	})
}
