package plan

import (
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"github.com/stretchr/testify/assert"
	"testing"
)

var rollingRequest = Request{
	Source: Source{
		API:      "a",
		Org:      "b",
		Space:    "c",
		Username: "d",
		Password: "e",
	},
	Params: Params{
		ManifestPath: "path/to/manifest.yml",
		AppPath:      "path/to/app",
		TestDomain:   "kehe.com",
		Vars: map[string]string{
			"VAR2": "bb",
			"VAR4": "cc",
		},
	},
}

func TestRollingDeployNormalApp(t *testing.T) {
	t.Run("Normal app", func(t *testing.T) {
		t.Run("No pre start", func(t *testing.T) {
			applicationManifest := manifest.Application{
				Name: "MyApp",
			}

			p := NewRollingDeployPlan().Plan(applicationManifest, rollingRequest, "")
			assert.Len(t, p, 3)
			assert.Equal(t, "cf push MyApp-CANDIDATE -f path/to/manifest.yml -p path/to/app --no-route --no-start", p[0].String())
			assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
			assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[2].String())
		})

		t.Run("Instances set", func(t *testing.T) {
			applicationManifest := manifest.Application{
				Name: "MyApp",
			}
			r := rollingRequest
			r.Params.Instances = "1"
			p := NewRollingDeployPlan().Plan(applicationManifest, r, "")
			assert.Len(t, p, 3)
			assert.Equal(t, "cf push MyApp-CANDIDATE -f path/to/manifest.yml -i 1 -p path/to/app --no-route --no-start", p[0].String())
			assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
			assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[2].String())
		})
		t.Run("With pre start", func(t *testing.T) {
			applicationManifest := manifest.Application{
				Name: "MyApp",
			}

			r := rollingRequest
			r.Params.PreStartCommand = "cf something; cf somethingElse"
			p := NewRollingDeployPlan().Plan(applicationManifest, r, "")

			assert.Len(t, p, 5)
			assert.Equal(t, "cf push MyApp-CANDIDATE -f path/to/manifest.yml -p path/to/app --no-route --no-start", p[0].String())
			assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
			assert.Equal(t, "cf something", p[2].String())
			assert.Equal(t, "cf somethingElse", p[3].String())
			assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[4].String())
		})
	})

	t.Run("Worker app", func(t *testing.T) {
		applicationManifest := manifest.Application{
			Name:    "MyApp",
			NoRoute: true,
		}

		p := NewRollingDeployPlan().Plan(applicationManifest, rollingRequest, "")
		assert.Len(t, p, 2)
		assert.Equal(t, "cf push MyApp-CANDIDATE -f path/to/manifest.yml -p path/to/app --no-route --no-start", p[0].String())
		assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[1].String())
	})
}
