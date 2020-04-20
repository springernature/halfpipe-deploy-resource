package plan

import (
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"github.com/stretchr/testify/assert"
	"testing"
)

var request = Request{
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

func TestNormalApp(t *testing.T) {
	t.Run("Normal app", func(t *testing.T) {
		t.Run("No pre start", func(t *testing.T) {
			applicationManifest := manifest.Application{
				Name: "MyApp",
			}

			p, _ := NewPushPlan().Plan(applicationManifest, request, "")
			assert.Len(t, p, 3)
			assert.Equal(t, p[0].String(), "cf push MyApp-CANDIDATE -f path/to/manifest.yml -p path/to/app --no-route --no-start")
			assert.Equal(t, p[1].String(), "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE")
			assert.Equal(t, p[2].String(), "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent")
		})
		t.Run("With pre start", func(t *testing.T) {
			applicationManifest := manifest.Application{
				Name: "MyApp",
			}

			r := request
			r.Params.PreStartCommand = "cf something; cf somethingElse"
			p, _ := NewPushPlan().Plan(applicationManifest, r, "")

			assert.Len(t, p, 5)
			assert.Equal(t, p[0].String(), "cf push MyApp-CANDIDATE -f path/to/manifest.yml -p path/to/app --no-route --no-start")
			assert.Equal(t, p[1].String(), "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE")
			assert.Equal(t, p[2].String(), "cf something")
			assert.Equal(t, p[3].String(), "cf somethingElse")
			assert.Equal(t, p[4].String(), "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent")
		})
	})

	t.Run("Worker app", func(t *testing.T) {
	})
}

func TestDocker(t *testing.T) {
	t.Run("Normal app", func(t *testing.T) {
	})

	t.Run("Worker app", func(t *testing.T) {
	})
}
