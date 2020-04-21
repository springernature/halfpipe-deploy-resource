package plan

import (
	"fmt"
	"github.com/google/uuid"
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

			p := NewPushPlan().Plan(applicationManifest, request, "")
			assert.Len(t, p, 3)
			assert.Equal(t, "cf push MyApp-CANDIDATE -f path/to/manifest.yml -p path/to/app --no-route --no-start", p[0].String())
			assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
			assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[2].String())
		})
		t.Run("With pre start", func(t *testing.T) {
			applicationManifest := manifest.Application{
				Name: "MyApp",
			}

			r := request
			r.Params.PreStartCommand = "cf something; cf somethingElse"
			p := NewPushPlan().Plan(applicationManifest, r, "")

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

		p := NewPushPlan().Plan(applicationManifest, request, "")
		assert.Len(t, p, 2)
		assert.Equal(t, "cf push MyApp-CANDIDATE -f path/to/manifest.yml -p path/to/app --no-route --no-start", p[0].String())
		assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[1].String())
	})
}

func TestDocker(t *testing.T) {
	t.Run("Normal app", func(t *testing.T) {
		applicationManifest := manifest.Application{
			Name: "MyApp",
			Docker: manifest.DockerInfo{
				Image: "wheep/whuup",
			},
		}

		r := request
		r.Params.DockerUsername = "asd"

		p := NewPushPlan().Plan(applicationManifest, r, "")
		assert.Len(t, p, 3)
		assert.Equal(t, "CF_DOCKER_PASSWORD=... cf push MyApp-CANDIDATE -f path/to/manifest.yml --docker-image wheep/whuup --docker-username asd", p[0].String())
		assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
		assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[2].String())
	})

	t.Run("Worker app", func(t *testing.T) {
		applicationManifest := manifest.Application{
			Name:    "MyApp",
			NoRoute: true,
			Docker: manifest.DockerInfo{
				Image: "wheep/whuup",
			},
		}

		r := request
		r.Params.DockerUsername = "kehe"

		p := NewPushPlan().Plan(applicationManifest, r, "")
		assert.Len(t, p, 2)
		assert.Equal(t, "CF_DOCKER_PASSWORD=... cf push MyApp-CANDIDATE -f path/to/manifest.yml --docker-image wheep/whuup --docker-username kehe", p[0].String())
		assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[1].String())
	})

	t.Run("Docker tag", func(t *testing.T) {
		t.Run("When it isn't set in the manifest, and we dont pass in an override", func(t *testing.T) {
			applicationManifest := manifest.Application{
				Name: "MyApp",
				Docker: manifest.DockerInfo{
					Image: "wheep/whuup",
				},
			}

			r := request
			r.Params.DockerUsername = "asd"

			p := NewPushPlan().Plan(applicationManifest, r, "")
			assert.Len(t, p, 3)
			assert.Equal(t, "CF_DOCKER_PASSWORD=... cf push MyApp-CANDIDATE -f path/to/manifest.yml --docker-image wheep/whuup --docker-username asd", p[0].String())
			assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
			assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[2].String())
		})

		t.Run("When it's set in the manifest, and we dont pass in an override", func(t *testing.T) {
			dockerTag := uuid.New().String()

			applicationManifest := manifest.Application{
				Name: "MyApp",
				Docker: manifest.DockerInfo{
					Image: fmt.Sprintf("wheep/whuup:%s", dockerTag),
				},
			}

			r := request
			r.Params.DockerUsername = "asd"

			p := NewPushPlan().Plan(applicationManifest, r, "")
			assert.Len(t, p, 3)
			assert.Equal(t, fmt.Sprintf("CF_DOCKER_PASSWORD=... cf push MyApp-CANDIDATE -f path/to/manifest.yml --docker-image wheep/whuup:%s --docker-username asd", dockerTag), p[0].String())
			assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
			assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[2].String())
		})

		t.Run("When it's not set in the manifest, and we pass in an override", func(t *testing.T) {
			dockerTag := uuid.New().String()

			applicationManifest := manifest.Application{
				Name: "MyApp",
				Docker: manifest.DockerInfo{
					Image: "wheep/whuup",
				},
			}

			r := request
			r.Params.DockerUsername = "asd"

			p := NewPushPlan().Plan(applicationManifest, r, dockerTag)
			assert.Len(t, p, 3)
			assert.Equal(t, fmt.Sprintf("CF_DOCKER_PASSWORD=... cf push MyApp-CANDIDATE -f path/to/manifest.yml --docker-image wheep/whuup:%s --docker-username asd", dockerTag), p[0].String())
			assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
			assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[2].String())
		})

		t.Run("When it's set in the manifest, and we pass in an override", func(t *testing.T) {
			dockerTag := uuid.New().String()

			applicationManifest := manifest.Application{
				Name: "MyApp",
				Docker: manifest.DockerInfo{
					Image: "wheep/whuup:somethingStatic",
				},
			}

			r := request
			r.Params.DockerUsername = "asd"

			p := NewPushPlan().Plan(applicationManifest, r, dockerTag)
			assert.Len(t, p, 3)
			assert.Equal(t, fmt.Sprintf("CF_DOCKER_PASSWORD=... cf push MyApp-CANDIDATE -f path/to/manifest.yml --docker-image wheep/whuup:%s --docker-username asd", dockerTag), p[0].String())
			assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
			assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[2].String())
		})

	})
}
