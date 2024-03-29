package plan

import (
	"fmt"
	"github.com/google/uuid"
	halfpipe_deploy_resource "github.com/springernature/halfpipe-deploy-resource"
	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

var request = config.Request{
	Source: config.Source{
		API:      "a",
		Org:      "b",
		Space:    "c",
		Username: "d",
		Password: "e",
	},
	Params: config.Params{
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

			var requestWithUnderscore = config.Request{
				Source: config.Source{
					API:      "a",
					Org:      "b",
					Space:    "this_is-a_space",
					Username: "d",
					Password: "e",
				},
				Params: config.Params{
					ManifestPath: "path/to/manifest.yml",
					AppPath:      "path/to/app",
					TestDomain:   "kehe.com",
					Vars: map[string]string{
						"VAR2": "bb",
						"VAR4": "cc",
					},
				},
			}

			applicationManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: My_App`).Applications[0]

			p := NewPushPlan().Plan(applicationManifest, requestWithUnderscore)
			assert.Len(t, p, 3)
			assert.Equal(t, "cf push My_App-CANDIDATE -f path/to/manifest.yml -p path/to/app --no-route --no-start", p[0].String())
			assert.Equal(t, "cf map-route My_App-CANDIDATE kehe.com -n My-App-this-is-a-space-CANDIDATE", p[1].String())
			assert.Equal(t, "cf start My_App-CANDIDATE || cf logs My_App-CANDIDATE --recent", p[2].String())
		})

		t.Run("Instances set", func(t *testing.T) {
			applicationManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: MyApp`).Applications[0]

			r := request
			r.Params.Instances = 1
			p := NewPushPlan().Plan(applicationManifest, r)
			assert.Len(t, p, 3)
			assert.Equal(t, "cf push MyApp-CANDIDATE -f path/to/manifest.yml -i 1 -p path/to/app --no-route --no-start", p[0].String())
			assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
			assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[2].String())
		})
		t.Run("With pre start", func(t *testing.T) {
			applicationManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: MyApp`).Applications[0]

			r := request
			r.Params.PreStartCommand = "cf something; cf somethingElse"
			p := NewPushPlan().Plan(applicationManifest, r)

			assert.Len(t, p, 5)
			assert.Equal(t, "cf push MyApp-CANDIDATE -f path/to/manifest.yml -p path/to/app --no-route --no-start", p[0].String())
			assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
			assert.Equal(t, "cf something", p[2].String())
			assert.Equal(t, "cf somethingElse", p[3].String())
			assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[4].String())
		})
	})

	t.Run("Worker app", func(t *testing.T) {
		applicationManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: MyApp
  no-route: True`).Applications[0]

		p := NewPushPlan().Plan(applicationManifest, request)
		assert.Len(t, p, 2)
		assert.Equal(t, "cf push MyApp-CANDIDATE -f path/to/manifest.yml -p path/to/app --no-route --no-start", p[0].String())
		assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[1].String())
	})
}

func TestDocker(t *testing.T) {
	t.Run("Normal app", func(t *testing.T) {
		applicationManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: MyApp
  docker:
    image: wheep/whuup`).Applications[0]

		r := request
		r.Params.DockerUsername = "asd"

		p := NewPushPlan().Plan(applicationManifest, r)
		assert.Len(t, p, 3)
		assert.Equal(t, "CF_DOCKER_PASSWORD=... cf push MyApp-CANDIDATE -f path/to/manifest.yml --docker-image wheep/whuup --docker-username asd --no-route --no-start", p[0].String())
		assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
		assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[2].String())
	})

	t.Run("Worker app", func(t *testing.T) {
		applicationManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: MyApp
  no-route: true
  docker:
    image: wheep/whuup`).Applications[0]

		r := request
		r.Params.DockerUsername = "kehe"

		p := NewPushPlan().Plan(applicationManifest, r)
		assert.Len(t, p, 2)
		assert.Equal(t, "CF_DOCKER_PASSWORD=... cf push MyApp-CANDIDATE -f path/to/manifest.yml --docker-image wheep/whuup --docker-username kehe --no-route --no-start", p[0].String())
		assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[1].String())
	})

	t.Run("Docker tag", func(t *testing.T) {
		t.Run("When it isn't set in the manifest, and we dont pass in an override", func(t *testing.T) {
			applicationManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: MyApp
  docker:
    image: wheep/whuup`).Applications[0]

			r := request
			r.Params.DockerUsername = "asd"

			p := NewPushPlan().Plan(applicationManifest, r)
			assert.Len(t, p, 3)
			assert.Equal(t, "CF_DOCKER_PASSWORD=... cf push MyApp-CANDIDATE -f path/to/manifest.yml --docker-image wheep/whuup --docker-username asd --no-route --no-start", p[0].String())
			assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
			assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[2].String())
		})

		t.Run("When it's set in the manifest, and we dont pass in an override", func(t *testing.T) {
			dockerTag := uuid.New().String()

			applicationManifest := halfpipe_deploy_resource.ParseManifest(fmt.Sprintf(`applications:
- name: MyApp
  docker:
    image: wheep/whuup:%s`, dockerTag)).Applications[0]

			r := request
			r.Params.DockerUsername = "asd"

			p := NewPushPlan().Plan(applicationManifest, r)
			assert.Len(t, p, 3)
			assert.Equal(t, fmt.Sprintf("CF_DOCKER_PASSWORD=... cf push MyApp-CANDIDATE -f path/to/manifest.yml --docker-image wheep/whuup:%s --docker-username asd --no-route --no-start", dockerTag), p[0].String())
			assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
			assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[2].String())
		})

		t.Run("When it's not set in the manifest, and we pass in an override", func(t *testing.T) {
			dockerTag := uuid.New().String()

			applicationManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: MyApp
  docker:
    image: wheep/whuup`).Applications[0]

			r := request
			r.Params.DockerUsername = "asd"
			r.Metadata.DockerTag = dockerTag

			p := NewPushPlan().Plan(applicationManifest, r)
			assert.Len(t, p, 3)
			assert.Equal(t, fmt.Sprintf("CF_DOCKER_PASSWORD=... cf push MyApp-CANDIDATE -f path/to/manifest.yml --docker-image wheep/whuup:%s --docker-username asd --no-route --no-start", dockerTag), p[0].String())
			assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
			assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[2].String())
		})

		t.Run("When it's set in the manifest, and we pass in an override", func(t *testing.T) {
			dockerTag := uuid.New().String()

			applicationManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: MyApp
  docker:
    image: wheep/whuup:somethingStatic`).Applications[0]

			r := request
			r.Params.DockerUsername = "asd"
			r.Metadata.DockerTag = dockerTag

			p := NewPushPlan().Plan(applicationManifest, r)
			assert.Len(t, p, 3)
			assert.Equal(t, fmt.Sprintf("CF_DOCKER_PASSWORD=... cf push MyApp-CANDIDATE -f path/to/manifest.yml --docker-image wheep/whuup:%s --docker-username asd --no-route --no-start", dockerTag), p[0].String())
			assert.Equal(t, "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE", p[1].String())
			assert.Equal(t, "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent", p[2].String())
		})

	})
}
