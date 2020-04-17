package plan

import (
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNormalApp(t *testing.T) {
	t.Run("Alles OK", func(t *testing.T) {
		space := "dev"
		manifestPath := "manifest.yml"
		appPath := "path/to/cool/app.jar"
		testDomain := "wicked.com"

		application := manifest.Application{
			Name: "my-app",
		}
		//
		expectedApplicationName := "my-app-CANDIDATE"         //helpers.CreateCandidateAppName(application.Name)
		expectedApplicationHostname := "my-app-dev-CANDIDATE" //helpers.CreateCandidateHostname(application.Name, space)

		expectedPlan := Plan{
			NewCfCommand("push", expectedApplicationName, "-f", manifestPath, "-p", appPath, "--no-route", "--no-start"),
			NewCfCommand("map-route", expectedApplicationName, testDomain, "-n", expectedApplicationHostname),
			NewCfCommand("start", expectedApplicationName),
		}

		request := Request{
			Source: Source{
				Space: space,
			},
			Params: Params{
				ManifestPath: manifestPath,
				AppPath:      appPath,
				TestDomain:   testDomain,
			},
		}

		pl, err := NewPushPlan(application, request).Plan()
		assert.Nil(t, err)
		assert.Equal(t, expectedPlan, pl)
	})
}

func TestDocker(t *testing.T) {

}
