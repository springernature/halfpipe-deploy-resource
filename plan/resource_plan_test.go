package plan

import (
	"fmt"
	"github.com/cloudfoundry-community/go-cfclient"
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

	push := NewPlanner(&ManifestReadWriteStub{readError: expectedError}, fs, []cfclient.AppSummary{})

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
	push := NewPlanner(&ManifestReadWriteStub{manifest: manifest, writeError: expectedError}, fs, []cfclient.AppSummary{})

	_, err := push.Plan(validRequest, concourseRoot)

	assert.Equal(t, expectedError, err)
}

func TestDoesntWriteManifestIfNotPush(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	concourseRoot := "/tmp/some/path"

	push := NewPlanner(&ManifestReadWriteStub{
		manifest:   manifest.Manifest{Applications: []manifest.Application{{}}},
		writeError: errors.New("should not happen")}, fs, []cfclient.AppSummary{})

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

	push := NewPlanner(manifestReadWrite, fs, []cfclient.AppSummary{})

	p, err := push.Plan(validRequest, "")

	assert.Nil(t, err)
	assert.Equal(t, expectedManifest, manifestReadWrite.savedManifest)
	assert.Len(t, p, 4)
	assert.Equal(t, p[0].String(), "cf login -a a -u d -p ******** -o b -s c")
	assert.Equal(t, p[1].String(), "cf push MyApp-CANDIDATE -f manifest.yml -p  --no-route --no-start")
	assert.Equal(t, p[2].String(), "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE")
	assert.Equal(t, p[3].String(), "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent")
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
	push := NewPlanner(&manifestReaderWriter, fs, []cfclient.AppSummary{})

	p, err := push.Plan(validRequest, "")

	assert.Nil(t, err)
	assert.Equal(t, expectedManifest, manifestReaderWriter.savedManifest)
	assert.Len(t, p, 4)
	assert.Equal(t, p[0].String(), "cf login -a a -u d -p ******** -o b -s c")
	assert.Equal(t, p[1].String(), "cf push MyApp-CANDIDATE -f manifest.yml -p  --no-route --no-start")
	assert.Equal(t, p[2].String(), "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE")
	assert.Equal(t, p[3].String(), "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent")
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
		push := NewPlanner(&manifestReaderWriter, fs, []cfclient.AppSummary{})

		request := validRequest
		request.Params.DockerUsername = "username"
		request.Params.DockerPassword = "superSecret"

		p, err := push.Plan(request, "")

		assert.Nil(t, err)
		assert.Len(t, p, 4)
		assert.Equal(t, p[0].String(), "cf login -a a -u d -p ******** -o b -s c")
		assert.Equal(t, p[1].String(), "CF_DOCKER_PASSWORD=... cf push MyApp-CANDIDATE -f manifest.yml --docker-image someCool/image:whoo --docker-username username")
		assert.Equal(t, p[2].String(), "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE")
		assert.Equal(t, p[3].String(), "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent")
		assert.NotEqual(t, p[1].String(), "appPath")

		assert.Equal(t, []string{fmt.Sprintf("CF_DOCKER_PASSWORD=%s", request.Params.DockerPassword)}, p[1].Env())
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
			push := NewPlanner(&manifestReaderWriter, fs, []cfclient.AppSummary{})

			request := validRequest
			request.Params.DockerUsername = "username"
			request.Params.DockerPassword = "superSecret"
			request.Params.DockerTag = tagFile

			p, err := push.Plan(request, "")

			assert.Nil(t, err)
			assert.Len(t, p, 4)
			assert.Equal(t, p[0].String(), "cf login -a a -u d -p ******** -o b -s c")
			assert.Equal(t, p[1].String(), fmt.Sprintf("CF_DOCKER_PASSWORD=... cf push MyApp-CANDIDATE -f manifest.yml --docker-image someCool/image:%s --docker-username username", tagContent))
			assert.Equal(t, p[2].String(), "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE")
			assert.Equal(t, p[3].String(), "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent")
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
			push := NewPlanner(&manifestReaderWriter, fs, []cfclient.AppSummary{})

			request := validRequest
			request.Params.DockerUsername = "username"
			request.Params.DockerPassword = "superSecret"
			request.Params.DockerTag = tagFile

			p, err := push.Plan(request, "")

			assert.Nil(t, err)
			assert.Len(t, p, 4)
			assert.Equal(t, p[0].String(), "cf login -a a -u d -p ******** -o b -s c")
			assert.Equal(t, p[1].String(), fmt.Sprintf("CF_DOCKER_PASSWORD=... cf push MyApp-CANDIDATE -f manifest.yml --docker-image someCool/image:%s --docker-username username", tagContent))
			assert.Equal(t, p[2].String(), "cf map-route MyApp-CANDIDATE kehe.com -n MyApp-c-CANDIDATE")
			assert.Equal(t, p[3].String(), "cf start MyApp-CANDIDATE || cf logs MyApp-CANDIDATE --recent")
		})
	})
}

func TestErrorsIfTheGitRefPathIsSpecifiedButDoesntExist(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	push := NewPlanner(&ManifestReadWriteStub{
		manifest: manifest.Manifest{[]manifest.Application{{}}},
	}, fs, []cfclient.AppSummary{})
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
	push := NewPlanner(&stub, fs, []cfclient.AppSummary{})

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
	push := NewPlanner(&stub, fs, []cfclient.AppSummary{})

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
