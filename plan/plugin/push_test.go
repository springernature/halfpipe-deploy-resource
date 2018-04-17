package plugin

import (
	"testing"

	"code.cloudfoundry.org/cli/plugin/models"
	"code.cloudfoundry.org/cli/util/manifest"
	"github.com/springernature/halfpipe-cf-plugin/plan"
	"github.com/stretchr/testify/assert"
)

func TestGivesBackAPushPlan(t *testing.T) {
	manifestPath := "path/to/manifest.yml"
	appPath := "path/to/app.jar"
	testDomain := "domain.com"
	space := "dev"

	application := manifest.Application{
		Name: "my-app",
	}

	expectedApplicationName := createCandidateAppName(application.Name)
	expectedApplicationHostname := createCandidateHostname(application.Name, space)

	expectedPlan := plan.Plan{
		plan.NewCfCommand("push", expectedApplicationName, "-f", manifestPath, "-p", appPath, "-n", expectedApplicationHostname, "-d", testDomain),
	}

	push := NewPushPlanner(newMockAppsGetter([]plugin_models.GetAppsModel{}, nil))

	commands, err := push.GetPlan(application, Request{
		ManifestPath: manifestPath,
		AppPath:      appPath,
		TestDomain:   testDomain,
		Space:        space,
	})

	assert.Nil(t, err)
	assert.Len(t, commands, 1)
	assert.Equal(t, expectedPlan, commands)
}

func TestGivesBackAPushPlanForWorkerApp(t *testing.T) {
	application := manifest.Application{
		Name:    "my-app",
		NoRoute: true,
	}
	expectedApplicationName := createCandidateAppName(application.Name)

	manifestPath := "path/to/manifest.yml"
	appPath := "path/to/app.jar"
	testDomain := "domain.com"

	expectedPlan := plan.Plan{
		plan.NewCfCommand("push", expectedApplicationName, "-f", manifestPath, "-p", appPath),
	}

	push := NewPushPlanner(newMockAppsGetter([]plugin_models.GetAppsModel{}, nil))

	commands, err := push.GetPlan(application, Request{
		ManifestPath: manifestPath,
		AppPath:      appPath,
		TestDomain:   testDomain,
	})

	assert.Nil(t, err)
	assert.Len(t, commands, 1)
	assert.Equal(t, expectedPlan, commands)
}

func TestFailsIfCandidateAppNameIsAlreadyInUse(t *testing.T) {

	candidateAppName := createCandidateAppName("app")

	apps := []plugin_models.GetAppsModel{
		{Name: candidateAppName},
	}

	pushPlanner := NewPushPlanner(newMockAppsGetter(apps, nil))

	ok, _ := pushPlanner.IsCFInAGoodState(candidateAppName, "blah", "blah")

	assert.False(t, ok)
}

func TestFailsIfCandidateRouteIsAlreadyInUse(t *testing.T) {
	appName := "my-app"
	candidateAppName := createCandidateAppName(appName)

	candidateHost := createCandidateHostname(appName, "dev")
	apps := []plugin_models.GetAppsModel{
		{Name: "app1", Routes: []plugin_models.GetAppsRouteSummary{{
			Host:   candidateHost,
			Domain: plugin_models.GetAppsDomainFields{Name: "testdomain.com"},
		}}},
	}

	pushPlanner := NewPushPlanner(newMockAppsGetter(apps, nil))

	ok, _ := pushPlanner.IsCFInAGoodState(candidateAppName, "testdomain.com", candidateHost)

	assert.False(t, ok)
}
