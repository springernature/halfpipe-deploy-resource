package manifest

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadManifestErrorsOutIfFileDoesntExist(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	mrw := NewManifestReadWrite(fs)

	manifest, err := mrw.ReadManifest("some/path")

	assert.Equal(t, Manifest{}, manifest)
	assert.Error(t, err)
}

func TestReadManifestReturnsManifestIfItExists(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	manifestPath := "path/to/manifest.yml"
	manifestStr := `---
applications:
- name: halfpipe-example-nodejs
  buildpack: https://github.com/cloudfoundry/nodejs-buildpack#v1.6.17
  command: node app.js
  memory: 1152MB
  instances: 2
  routes:
  - route: halfpipe-example-nodejs.dev.private.springernature.io
`

	expectedManifest := Manifest{
		Applications: []Application{
			{
				Name:      "halfpipe-example-nodejs",
				Buildpack: "https://github.com/cloudfoundry/nodejs-buildpack#v1.6.17",
				Command:   "node app.js",
				Memory:    "1152MB",
				Instances: 2,
				Routes: []Route{
					{Route: "halfpipe-example-nodejs.dev.private.springernature.io"},
				},
			},
		},
	}

	fs.WriteFile(manifestPath, []byte(manifestStr), 0666)

	mrw := NewManifestReadWrite(fs)

	manifest, err := mrw.ReadManifest(manifestPath)

	assert.Equal(t, expectedManifest, manifest)
	assert.Nil(t, err)
}

func TestWriteManifest(t *testing.T) {

	expectedManifest := `applications:
- name: halfpipe-example-nodejs
  buildpack: https://github.com/cloudfoundry/nodejs-buildpack#v1.6.17
  command: node app.js
  instances: 2
  memory: 1152MB
  routes:
  - route: halfpipe-example-nodejs.dev.private.springernature.io
`

	application := Application{
		Name:      "halfpipe-example-nodejs",
		Buildpack: "https://github.com/cloudfoundry/nodejs-buildpack#v1.6.17",
		Command:   "node app.js",
		Memory:    "1152MB",
		Instances: 2,
		Routes: []Route{
			{Route: "halfpipe-example-nodejs.dev.private.springernature.io"},
		},
	}

	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	manifestPath := "path/to/manifest.yml"

	mrw := NewManifestReadWrite(fs)

	err := mrw.WriteManifest(manifestPath, application)
	assert.Nil(t, err)

	fileBytes, err := fs.ReadFile(manifestPath)
	assert.Nil(t, err)

	assert.Equal(t, expectedManifest, string(fileBytes))
}
