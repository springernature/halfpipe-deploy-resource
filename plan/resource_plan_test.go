package plan

import (
	"fmt"
	"testing"

	"errors"
	"github.com/google/uuid"
	"github.com/spf13/afero"
	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"github.com/stretchr/testify/assert"
	"path"
)

var validRequest = Request{
	Source: Source{
		API:      "a",
		Org:      "b",
		Space:    "c",
		Username: "d",
		Password: "e",
	},
	Params: Params{
		ManifestPath: "manifest.yml",
		AppPath:      "",
		TestDomain:   "kehe.com",
		Command:      config.PUSH,
		Vars: map[string]string{
			"VAR2": "bb",
			"VAR4": "cc",
		},
	},
}

type ManifestReadWriteStub struct {
	manifest      manifest.Manifest
	readError     error
	writeError    error
	savedManifest manifest.Manifest
}

func (m *ManifestReadWriteStub) ReadManifest(path string) (manifest.Manifest, error) {
	return m.manifest, m.readError
}

func (m *ManifestReadWriteStub) WriteManifest(path string, application manifest.Application) error {
	m.savedManifest = manifest.Manifest{
		Applications: []manifest.Application{application},
	}

	return m.writeError
}

func TestReturnsErrorIfWeFailToReadManifest(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	expectedError := errors.New("Shiied")

	concourseRoot := "/tmp/some/path"

	push := NewPlanner(&ManifestReadWriteStub{readError: expectedError}, fs)

	_, err := push.Plan(validRequest, concourseRoot)
	assert.Equal(t, expectedError, err)
}

func TestReturnsErrorIfWeFailToWriteManifest(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	expectedError := errors.New("Shiied")

	concourseRoot := "/tmp/some/path"

	manifest := manifest.Manifest{
		Applications: []manifest.Application{{}},
	}
	push := NewPlanner(&ManifestReadWriteStub{manifest: manifest, writeError: expectedError}, fs)

	_, err := push.Plan(validRequest, concourseRoot)

	assert.Equal(t, expectedError, err)
}

func TestDoesntWriteManifestIfNotPush(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	concourseRoot := "/tmp/some/path"

	push := NewPlanner(
		&ManifestReadWriteStub{
			readError:  errors.New("should not happen"),
			writeError: errors.New("should not happen")},
		fs)

	validPromoteRequest := Request{
		Source: Source{
			API:      "a",
			Org:      "b",
			Space:    "c",
			Username: "d",
			Password: "e",
		},
		Params: Params{
			ManifestPath: "manifest.yml",
			AppPath:      "",
			TestDomain:   "kehe.com",
			Command:      config.PROMOTE,
			Vars: map[string]string{
				"VAR2": "bb",
				"VAR4": "cc",
			},
		},
	}

	_, err := push.Plan(validPromoteRequest, concourseRoot)

	assert.Nil(t, err)
}

func TestGivesACorrectPlanWhenManifestDoesNotHaveAnyEnvironmentVariables(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	applicationManifest := manifest.Manifest{
		Applications: []manifest.Application{
			{Name: "MyApp"},
		},
	}
	expectedManifest := manifest.Manifest{
		Applications: []manifest.Application{
			{
				Name:                 "MyApp",
				EnvironmentVariables: validRequest.Params.Vars,
			},
		},
	}

	manifestReadWrite := &ManifestReadWriteStub{manifest: applicationManifest}

	push := NewPlanner(manifestReadWrite, fs)

	p, err := push.Plan(validRequest, "")

	assert.Nil(t, err)
	assert.Equal(t, expectedManifest, manifestReadWrite.savedManifest)
	assert.Len(t, p, 2)
	assert.Contains(t, p[0].String(), "cf login")
	assert.Contains(t, p[1].String(), "cf halfpipe-push")
}

func TestGivesACorrectPlanThatAlsoOverridesVariablesInManifest(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	applicationManifest := manifest.Manifest{
		Applications: []manifest.Application{
			{
				Name: "MyApp",
				EnvironmentVariables: map[string]string{
					"VAR1": "a",
					"VAR2": "b",
					"VAR3": "c",
				},
			},
		},
	}

	expectedManifest := manifest.Manifest{
		Applications: []manifest.Application{
			{
				Name: "MyApp",
				EnvironmentVariables: map[string]string{
					"VAR1": "a",
					"VAR2": "bb",
					"VAR3": "c",
					"VAR4": "cc",
				},
			},
		},
	}

	manifestReaderWriter := ManifestReadWriteStub{manifest: applicationManifest}
	push := NewPlanner(&manifestReaderWriter, fs)

	p, err := push.Plan(validRequest, "")

	assert.Nil(t, err)
	assert.Equal(t, expectedManifest, manifestReaderWriter.savedManifest)
	assert.Len(t, p, 2)
	assert.Contains(t, p[0].String(), "cf login")
	assert.Contains(t, p[1].String(), "cf halfpipe-push")
}

func TestGivesACorrectPlanWhenDockerImageSpecifiedInManifest(t *testing.T) {
	t.Run("Without tag", func(t *testing.T) {
		fs := afero.Afero{Fs: afero.NewMemMapFs()}
		applicationManifest := manifest.Manifest{
			Applications: []manifest.Application{
				{
					Name: "MyApp",
					EnvironmentVariables: map[string]string{
						"VAR1": "a",
						"VAR2": "b",
						"VAR3": "c",
					},
					Docker: manifest.DockerInfo{
						Image: "someCool/image:whoo",
					},
				},
			},
		}

		manifestReaderWriter := ManifestReadWriteStub{manifest: applicationManifest}
		push := NewPlanner(&manifestReaderWriter, fs)

		request := validRequest
		request.Params.DockerUsername = "username"
		request.Params.DockerPassword = "superSecret"

		p, err := push.Plan(request, "")

		assert.Nil(t, err)
		assert.Len(t, p, 2)
		assert.Contains(t, p[0].String(), "cf login")
		assert.Contains(t, p[1].String(), "CF_DOCKER_PASSWORD=... cf halfpipe-push")
		assert.Contains(t, p[1].String(), fmt.Sprintf("-dockerUsername %s", request.Params.DockerUsername))
		assert.Contains(t, p[1].String(), fmt.Sprintf("-dockerImage %s", applicationManifest.Applications[0].Docker.Image))
		assert.NotContains(t, p[1].String(), "appPath")

		assert.Equal(t, p[1].Env(), []string{fmt.Sprintf("CF_DOCKER_PASSWORD=%s", request.Params.DockerPassword)})
	})

	t.Run("With tag", func(t *testing.T) {
		t.Run("When image in manifest doesnt specify version", func(t *testing.T) {
			fs := afero.Afero{Fs: afero.NewMemMapFs()}

			tagFile := "path/To/TagFile"
			tagContent := uuid.New().String()
			fs.WriteFile(tagFile, []byte(tagContent), 0777)

			applicationManifest := manifest.Manifest{
				Applications: []manifest.Application{
					{
						Name: "MyApp",
						EnvironmentVariables: map[string]string{
							"VAR1": "a",
							"VAR2": "b",
							"VAR3": "c",
						},
						Docker: manifest.DockerInfo{
							Image: "someCool/image",
						},
					},
				},
			}

			manifestReaderWriter := ManifestReadWriteStub{manifest: applicationManifest}
			push := NewPlanner(&manifestReaderWriter, fs)

			request := validRequest
			request.Params.DockerUsername = "username"
			request.Params.DockerPassword = "superSecret"
			request.Params.DockerTag = tagFile

			p, err := push.Plan(request, "")

			assert.Nil(t, err)
			assert.Len(t, p, 2)
			assert.Contains(t, p[0].String(), "cf login")
			assert.Contains(t, p[1].String(), "CF_DOCKER_PASSWORD=... cf halfpipe-push")
			assert.Contains(t, p[1].String(), fmt.Sprintf("-dockerImage %s", fmt.Sprintf("%s:%s", applicationManifest.Applications[0].Docker.Image, tagContent)))
		})

		t.Run("When image in manifest specifies version", func(t *testing.T) {
			fs := afero.Afero{Fs: afero.NewMemMapFs()}

			tagFile := "path/To/TagFile"
			tagContent := uuid.New().String()
			fs.WriteFile(tagFile, []byte(tagContent), 0777)

			applicationManifest := manifest.Manifest{
				Applications: []manifest.Application{
					{
						Name: "MyApp",
						EnvironmentVariables: map[string]string{
							"VAR1": "a",
							"VAR2": "b",
							"VAR3": "c",
						},
						Docker: manifest.DockerInfo{
							Image: "someCool/image:someTag",
						},
					},
				},
			}

			manifestReaderWriter := ManifestReadWriteStub{manifest: applicationManifest}
			push := NewPlanner(&manifestReaderWriter, fs)

			request := validRequest
			request.Params.DockerUsername = "username"
			request.Params.DockerPassword = "superSecret"
			request.Params.DockerTag = tagFile

			p, err := push.Plan(request, "")

			assert.Nil(t, err)
			assert.Len(t, p, 2)
			assert.Contains(t, p[0].String(), "cf login")
			assert.Contains(t, p[1].String(), "CF_DOCKER_PASSWORD=... cf halfpipe-push")
			assert.Contains(t, p[1].String(), fmt.Sprintf("-dockerImage %s", fmt.Sprintf("%s:%s", "someCool/image", tagContent)))
		})
	})
}

func TestErrorsIfTheGitRefPathIsSpecifiedButDoesntExist(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	push := NewPlanner(&ManifestReadWriteStub{
		manifest: manifest.Manifest{[]manifest.Application{{}}},
	}, fs)
	request := Request{
		Source: Source{
			API:      "a",
			Org:      "b",
			Space:    "c",
			Username: "d",
			Password: "e",
		},
		Params: Params{
			ManifestPath: "manifest.yml",
			GitRefPath:   "git/.git/ref",
			AppPath:      "",
			TestDomain:   "kehe.com",
			Command:      config.PUSH,
		},
	}
	_, err := push.Plan(request, "/some/path")

	assert.Error(t, err)
}

func TestPutsGitRefInTheManifest(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	concourseRoot := "/some/path"
	gitRefPath := "git/.git/ref"
	gitRef := "wiiiie\n"
	fs.WriteFile(path.Join(concourseRoot, gitRefPath), []byte(gitRef), 0700)

	applicationManifest := manifest.Manifest{
		Applications: []manifest.Application{
			{
				Name: "MyApp",
				EnvironmentVariables: map[string]string{
					"VAR1": "a",
					"VAR2": "b",
					"VAR3": "c",
				},
			},
		},
	}

	stub := ManifestReadWriteStub{manifest: applicationManifest}
	push := NewPlanner(&stub, fs)

	request := Request{
		Source: Source{
			API:      "a",
			Org:      "b",
			Space:    "c",
			Username: "d",
			Password: "e",
		},
		Params: Params{
			ManifestPath: "manifest.yml",
			GitRefPath:   gitRefPath,
			AppPath:      "",
			TestDomain:   "kehe.com",
			Command:      config.PUSH,
		},
	}

	_, err := push.Plan(request, concourseRoot)

	assert.Nil(t, err)
	assert.Equal(t, stub.savedManifest.Applications[0].EnvironmentVariables["GIT_REVISION"], "wiiiie")
}

func TestPutsBuildVersionInTheManifest(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	concourseRoot := "/some/path"
	gitRefPath := "git/.git/ref"
	buildVersionPath := "version/version"
	gitRef := "wiiiie\n"
	buildVersion := "1.1.0\n"
	fs.WriteFile(path.Join(concourseRoot, gitRefPath), []byte(gitRef), 0700)
	fs.WriteFile(path.Join(concourseRoot, buildVersionPath), []byte(buildVersion), 0700)

	applicationManifest := manifest.Manifest{
		Applications: []manifest.Application{
			{
				Name: "MyApp",
				EnvironmentVariables: map[string]string{
					"VAR1": "a",
					"VAR2": "b",
					"VAR3": "c",
				},
			},
		},
	}

	stub := ManifestReadWriteStub{manifest: applicationManifest}
	push := NewPlanner(&stub, fs)

	request := Request{
		Source: Source{
			API:      "a",
			Org:      "b",
			Space:    "c",
			Username: "d",
			Password: "e",
		},
		Params: Params{
			ManifestPath:     "manifest.yml",
			GitRefPath:       gitRefPath,
			BuildVersionPath: buildVersionPath,
			AppPath:          "",
			TestDomain:       "kehe.com",
			Command:          config.PUSH,
		},
	}

	_, err := push.Plan(request, concourseRoot)

	assert.Nil(t, err)
	assert.Equal(t, stub.savedManifest.Applications[0].EnvironmentVariables["GIT_REVISION"], "wiiiie")
	assert.Equal(t, stub.savedManifest.Applications[0].EnvironmentVariables["BUILD_VERSION"], "1.1.0")
}

func TestAddsTimoutIfSpecified(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	var requestWithTimeout = Request{
		Source: Source{
			Space: "dev",
		},
		Params: Params{
			Command:      config.PUSH,
			ManifestPath: "manifest.yml",
			AppPath:      ".",
			TestDomain:   "domain.com",
			Timeout:      "1m",
		},
	}

	applicationManifest := manifest.Manifest{
		Applications: []manifest.Application{
			{Name: "MyApp"},
		},
	}

	manifestReadWrite := &ManifestReadWriteStub{manifest: applicationManifest}

	push := NewPlanner(manifestReadWrite, fs)

	p, err := push.Plan(requestWithTimeout, "")

	assert.Nil(t, err)
	assert.Len(t, p, 2)
	assert.Contains(t, p[0].String(), "cf login")
	assert.Equal(t, "cf halfpipe-push -manifestPath manifest.yml -testDomain domain.com -appPath . -timeout 1m || cf logs MyApp-CANDIDATE --recent", p[1].String())
}

func TestAddsPreStartCommandIfSpecified(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	var requestWithPreStartCommand = Request{
		Source: Source{
			Space: "dev",
		},
		Params: Params{
			Command:         config.PUSH,
			ManifestPath:    "manifest.yml",
			AppPath:         ".",
			TestDomain:      "domain.com",
			Timeout:         "1m",
			PreStartCommand: "cf something \"or other\"",
		},
	}

	applicationManifest := manifest.Manifest{
		Applications: []manifest.Application{
			{Name: "MyApp"},
		},
	}

	manifestReadWrite := &ManifestReadWriteStub{manifest: applicationManifest}

	push := NewPlanner(manifestReadWrite, fs)

	p, err := push.Plan(requestWithPreStartCommand, "")

	assert.Nil(t, err)
	assert.Len(t, p, 2)
	assert.Contains(t, p[0].String(), "cf login")
	assert.Equal(t, `cf halfpipe-push -manifestPath manifest.yml -testDomain domain.com -appPath . -preStartCommand "cf something \"or other\"" -timeout 1m || cf logs MyApp-CANDIDATE --recent`, p[1].String())
}

func TestGivesACorrectPlanWhenInstancesIsSet(t *testing.T) {
	request := Request{
		Source: Source{
			API:      "a",
			Org:      "b",
			Space:    "c",
			Username: "d",
			Password: "e",
		},
		Params: Params{
			ManifestPath: "manifest.yml",
			AppPath:      "",
			TestDomain:   "kehe.com",
			Command:      config.PUSH,
			Instances:    "1337",
		},
	}

	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	applicationManifest := manifest.Manifest{
		Applications: []manifest.Application{
			{Name: "MyApp"},
		},
	}

	manifestReadWrite := &ManifestReadWriteStub{manifest: applicationManifest}

	push := NewPlanner(manifestReadWrite, fs)

	p, err := push.Plan(request, "")

	assert.Nil(t, err)
	assert.Len(t, p, 2)
	assert.Contains(t, p[0].String(), "cf login")
	assert.Contains(t, p[1].String(), "cf halfpipe-push")
	assert.Contains(t, p[1].String(), "-instances 1337")
}

func TestGivesACorrectRollingDeployPlan(t *testing.T) {
	request := Request{
		Source: Source{
			API:      "a",
			Org:      "b",
			Space:    "c",
			Username: "d",
			Password: "e",
		},
		Params: Params{
			ManifestPath: "manifest.yml",
			AppPath:      "some/cool/path",
			TestDomain:   "kehe.com",
			Command:      config.DEPLOY_ROLLING,
		},
	}

	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	applicationManifest := manifest.Manifest{
		Applications: []manifest.Application{
			{Name: "MyApp"},
		},
	}

	manifestReadWrite := &ManifestReadWriteStub{manifest: applicationManifest}

	push := NewPlanner(manifestReadWrite, fs)

	p, err := push.Plan(request, "")

	assert.Nil(t, err)
	assert.Len(t, p, 2)
	assert.Contains(t, p[0].String(), "cf login")
	assert.Equal(t, "cf push --manifest manifest.yml --path some/cool/path --strategy rolling", p[1].String())
}

func TestRolling(t *testing.T) {
}
