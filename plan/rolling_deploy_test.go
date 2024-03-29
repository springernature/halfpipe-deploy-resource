package plan

import (
	"fmt"
	"github.com/google/uuid"
	halfpipe_deploy_resource "github.com/springernature/halfpipe-deploy-resource"
	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

var rollingRequest = config.Request{
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

func TestRollingDeployNormalApp(t *testing.T) {
	t.Run("Normal app", func(t *testing.T) {
		t.Run("No pre start", func(t *testing.T) {
			applicationManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: MyApp`).Applications[0]

			p := NewRollingDeployPlan().Plan(applicationManifest, rollingRequest)
			assert.Len(t, p, 1)
			assert.Equal(t, "cf push --manifest path/to/manifest.yml --strategy rolling --path path/to/app || cf logs MyApp --recent", p[0].String())
		})
	})
}

func TestRollingDeployDocker(t *testing.T) {
	t.Run("Normal app", func(t *testing.T) {
		applicationManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: MyApp
  docker:
    image: wheep/whuup
`).Applications[0]

		r := request
		r.Params.DockerUsername = "asd"

		p := NewRollingDeployPlan().Plan(applicationManifest, r)
		assert.Len(t, p, 1)
		assert.Equal(t, "CF_DOCKER_PASSWORD=... cf push --manifest path/to/manifest.yml --strategy rolling --docker-image wheep/whuup --docker-username asd || cf logs MyApp --recent", p[0].String())
	})

	t.Run("Docker tag", func(t *testing.T) {
		t.Run("When it isn't set in the manifest, and we dont pass in an override", func(t *testing.T) {
			applicationManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: MyApp
  docker:
    image: wheep/whuup
`).Applications[0]

			r := request
			r.Params.DockerUsername = "asd"

			p := NewRollingDeployPlan().Plan(applicationManifest, r)
			assert.Len(t, p, 1)
			assert.Equal(t, "CF_DOCKER_PASSWORD=... cf push --manifest path/to/manifest.yml --strategy rolling --docker-image wheep/whuup --docker-username asd || cf logs MyApp --recent", p[0].String())
		})

		t.Run("When it's set in the manifest, and we dont pass in an override", func(t *testing.T) {
			dockerTag := uuid.New().String()

			applicationManifest := halfpipe_deploy_resource.ParseManifest(fmt.Sprintf(`applications:
- name: MyApp
  docker:
    image: wheep/whuup:%s`, dockerTag)).Applications[0]

			r := request
			r.Params.DockerUsername = "asd"

			p := NewRollingDeployPlan().Plan(applicationManifest, r)
			assert.Len(t, p, 1)
			assert.Equal(t, fmt.Sprintf("CF_DOCKER_PASSWORD=... cf push --manifest path/to/manifest.yml --strategy rolling --docker-image wheep/whuup:%s --docker-username asd || cf logs MyApp --recent", dockerTag), p[0].String())
		})

		t.Run("When it's not set in the manifest, and we pass in an override", func(t *testing.T) {
			dockerTag := uuid.New().String()

			applicationManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: MyApp
  docker:
    image: wheep/whuup
`).Applications[0]

			r := request
			r.Params.DockerUsername = "asd"
			r.Metadata.DockerTag = dockerTag

			p := NewRollingDeployPlan().Plan(applicationManifest, r)
			assert.Len(t, p, 1)
			assert.Equal(t, fmt.Sprintf("CF_DOCKER_PASSWORD=... cf push --manifest path/to/manifest.yml --strategy rolling --docker-image wheep/whuup:%s --docker-username asd || cf logs MyApp --recent", dockerTag), p[0].String())

		})

		t.Run("When it's set in the manifest, and we pass in an override", func(t *testing.T) {
			dockerTag := uuid.New().String()

			applicationManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: MyApp
  docker:
    image: wheep/whuup:somethingStatic
`).Applications[0]

			r := request
			r.Params.DockerUsername = "asd"
			r.Metadata.DockerTag = dockerTag

			p := NewRollingDeployPlan().Plan(applicationManifest, r)
			assert.Len(t, p, 1)
			assert.Equal(t, fmt.Sprintf("CF_DOCKER_PASSWORD=... cf push --manifest path/to/manifest.yml --strategy rolling --docker-image wheep/whuup:%s --docker-username asd || cf logs MyApp --recent", dockerTag), p[0].String())
		})

	})
}
