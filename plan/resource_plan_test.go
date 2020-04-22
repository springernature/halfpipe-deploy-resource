package plan

import (
	"errors"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"path"
	"testing"

	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
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
		ManifestPath:     "manifest.yml",
		AppPath:          "",
		TestDomain:       "kehe.com",
		Command:          config.PUSH,
		GitRefPath:       "gitRefPath",
		BuildVersionPath: "buildVersionPath",
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

	readPath  string
	writePath string
}

func (m *ManifestReadWriteStub) ReadManifest(path string) (manifest.Manifest, error) {
	m.readPath = path
	return m.manifest, m.readError
}

func (m *ManifestReadWriteStub) WriteManifest(path string, application manifest.Application) error {
	m.writePath = path
	m.savedManifest = manifest.Manifest{
		Applications: []manifest.Application{application},
	}

	return m.writeError
}

//
var concourseRoot = "/tmp/some/path"

func TestErrorsReadingAppManifest(t *testing.T) {
	expectedErr := errors.New("blurgh")
	expectedPath := path.Join(concourseRoot, validRequest.Params.ManifestPath)
	manifestReader := ManifestReadWriteStub{readError: expectedErr}

	planner := NewPlanner(&manifestReader, afero.Afero{}, nil, nil, nil)

	_, err := planner.Plan(validRequest, concourseRoot, nil)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, expectedPath, manifestReader.readPath)
}

func TestErrorsWhenWeFailToReadGitRef(t *testing.T) {
	manifestReader := ManifestReadWriteStub{
		manifest: manifest.Manifest{
			Applications: []manifest.Application{
				{},
			},
		},
	}

	planner := NewPlanner(&manifestReader, afero.Afero{Fs: afero.NewMemMapFs()}, nil, nil, nil)

	_, err := planner.Plan(validRequest, concourseRoot, nil)
	assert.Equal(t, "open /tmp/some/path/gitRefPath: file does not exist", err.Error())
}

func TestErrorsWhenWeFailToReadBuildVersion(t *testing.T) {
	manifestReader := ManifestReadWriteStub{
		manifest: manifest.Manifest{
			Applications: []manifest.Application{
				{},
			},
		},
	}

	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	fs.WriteFile(path.Join(concourseRoot, validRequest.Params.GitRefPath), []byte(""), 0777)
	planner := NewPlanner(&manifestReader, fs, nil, nil, nil)

	_, err := planner.Plan(validRequest, concourseRoot, nil)
	assert.Equal(t, "open /tmp/some/path/buildVersionPath: file does not exist", err.Error())
}

func TestErrorsWhenSavingManifestWithUpdatedVars(t *testing.T) {
	expectedErr := errors.New("blurgh")
	expectedPath := path.Join(concourseRoot, validRequest.Params.ManifestPath)
	manifestReader := ManifestReadWriteStub{
		manifest: manifest.Manifest{
			Applications: []manifest.Application{
				{},
			},
		},
		writeError: expectedErr,
	}

	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	fs.WriteFile(path.Join(concourseRoot, validRequest.Params.GitRefPath), []byte(""), 0777)
	fs.WriteFile(path.Join(concourseRoot, validRequest.Params.BuildVersionPath), []byte(""), 0777)

	planner := NewPlanner(&manifestReader, fs, nil, nil, nil)

	_, err := planner.Plan(validRequest, concourseRoot, nil)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, expectedPath, manifestReader.readPath)
	assert.Equal(t, expectedPath, manifestReader.writePath)
}

func TestErrorsWhenReadingDockerTag(t *testing.T) {
	expectedErr := errors.New("open /tmp/some/path/some/path/to/a/DockerTagFile: file does not exist")
	expectedPath := path.Join(concourseRoot, validRequest.Params.ManifestPath)
	manifestReader := ManifestReadWriteStub{
		manifest: manifest.Manifest{
			Applications: []manifest.Application{
				{
					Docker: manifest.DockerInfo{
						Image: "yo/brawh",
					},
				},
			},
		},
	}
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	fs.WriteFile(path.Join(concourseRoot, validRequest.Params.GitRefPath), []byte(""), 0777)
	fs.WriteFile(path.Join(concourseRoot, validRequest.Params.BuildVersionPath), []byte(""), 0777)

	planner := NewPlanner(&manifestReader, fs, nil, nil, nil)

	r := validRequest
	r.Params.DockerTag = "/some/path/to/a/DockerTagFile"

	_, err := planner.Plan(r, concourseRoot, nil)

	assert.Equal(t, expectedErr.Error(), err.Error())
	assert.Equal(t, expectedPath, manifestReader.readPath)
	assert.Equal(t, expectedPath, manifestReader.writePath)
}

type fakePushPlanner struct {
	plan      Plan
	dockerTag string
}

type fakePromotePlanner struct {
	plan Plan
}

type fakeCleanupPlanner struct {
	plan Plan
}

func (f fakePromotePlanner) Plan(manifest manifest.Application, request Request, summary []cfclient.AppSummary) (pl Plan) {
	return f.plan
}

func (f fakeCleanupPlanner) Plan(manifest manifest.Application, request Request, summary []cfclient.AppSummary) (pl Plan) {
	return f.plan
}

func (f *fakePushPlanner) Plan(manifest manifest.Application, request Request, dockerTag string) (pl Plan) {
	f.dockerTag = dockerTag
	return f.plan
}

func TestCallsOutToCorrectPlanner(t *testing.T) {
	t.Run("Push planner", func(t *testing.T) {

		t.Run("Normal app", func(t *testing.T) {
			expectedPath := path.Join(concourseRoot, validRequest.Params.ManifestPath)

			expectedManifest := manifest.Manifest{
				Applications: []manifest.Application{
					{
						EnvironmentVariables: validRequest.Params.Vars,
					},
				},
			}

			manifestReader := ManifestReadWriteStub{
				manifest: manifest.Manifest{
					Applications: []manifest.Application{
						{},
					},
				},
			}

			planner := NewPlanner(&manifestReader, afero.Afero{Fs: afero.NewMemMapFs()}, &fakePushPlanner{
				plan: Plan{
					NewCfCommand("yay"),
				},
			}, nil, nil)

			r := validRequest
			r.Params.BuildVersionPath = ""
			r.Params.GitRefPath = ""
			p, err := planner.Plan(r, concourseRoot, nil)

			assert.NoError(t, err)

			assert.Len(t, p, 2)
			assert.Equal(t, "cf login -a a -u d -p ******** -o b -s c", p[0].String())
			assert.Equal(t, "cf yay", p[1].String())
			assert.Equal(t, expectedPath, manifestReader.readPath)
			assert.Equal(t, expectedPath, manifestReader.writePath)
			assert.Equal(t, expectedManifest, manifestReader.savedManifest)
		})

		t.Run("Docker app", func(t *testing.T) {
			expectedPath := path.Join(concourseRoot, validRequest.Params.ManifestPath)

			expectedManifest := manifest.Manifest{
				Applications: []manifest.Application{
					{
						Docker: manifest.DockerInfo{
							Image: "yo/sup",
						},
						EnvironmentVariables: validRequest.Params.Vars,
					},
				},
			}

			manifestReader := ManifestReadWriteStub{
				manifest: manifest.Manifest{
					Applications: []manifest.Application{
						{
							Docker: manifest.DockerInfo{
								Image: "yo/sup",
							},
						},
					},
				},
			}
			fs := afero.Afero{Fs: afero.NewMemMapFs()}

			r := validRequest
			r.Params.DockerTag = "something"
			r.Params.GitRefPath = ""
			r.Params.BuildVersionPath = ""
			fullDockerTagPath := path.Join(concourseRoot, r.Params.DockerTag)

			expectedDockerTag := "this is a cool uuid"
			fs.WriteFile(fullDockerTagPath, []byte(expectedDockerTag), 0777)

			pushPlanner := fakePushPlanner{
				plan: Plan{
					NewCfCommand("yay"),
				},
			}
			planner := NewPlanner(&manifestReader, fs, &pushPlanner, nil, nil)

			p, err := planner.Plan(r, concourseRoot, nil)

			assert.NoError(t, err)

			assert.Len(t, p, 2)
			assert.Equal(t, "cf login -a a -u d -p ******** -o b -s c", p[0].String())
			assert.Equal(t, "cf yay", p[1].String())
			assert.Equal(t, expectedPath, manifestReader.readPath)
			assert.Equal(t, expectedPath, manifestReader.writePath)
			assert.Equal(t, expectedManifest, manifestReader.savedManifest)
			assert.Equal(t, expectedDockerTag, pushPlanner.dockerTag)
		})
	})

	t.Run("Promote planner", func(t *testing.T) {
		manifestReader := ManifestReadWriteStub{
			manifest: manifest.Manifest{
				Applications: []manifest.Application{
					{},
				},
			},
		}

		planner := NewPlanner(&manifestReader, afero.Afero{Fs: afero.NewMemMapFs()}, nil, fakePromotePlanner{
			plan: Plan{
				NewCfCommand("yay"),
			},
		}, nil)

		r := validRequest
		r.Params.Command = config.PROMOTE

		p, err := planner.Plan(r, concourseRoot, nil)

		assert.NoError(t, err)

		assert.Len(t, p, 2)
		assert.Equal(t, "cf login -a a -u d -p ******** -o b -s c", p[0].String())
		assert.Equal(t, "cf yay", p[1].String())
	})

	t.Run("Cleanup planner", func(t *testing.T) {
		manifestReader := ManifestReadWriteStub{
			manifest: manifest.Manifest{
				Applications: []manifest.Application{
					{},
				},
			},
		}

		planner := NewPlanner(&manifestReader, afero.Afero{Fs: afero.NewMemMapFs()}, nil, nil, fakePromotePlanner{
			plan: Plan{
				NewCfCommand("yay"),
			},
		})

		t.Run("Works with cleanup command", func(t *testing.T) {
			r := validRequest
			r.Params.Command = config.CLEANUP

			p, err := planner.Plan(r, concourseRoot, nil)

			assert.NoError(t, err)

			assert.Len(t, p, 2)
			assert.Equal(t, "cf login -a a -u d -p ******** -o b -s c", p[0].String())
			assert.Equal(t, "cf yay", p[1].String())
		})

		t.Run("Works with a delete command", func(t *testing.T) {
			r := validRequest
			r.Params.Command = config.DELETE

			p, err := planner.Plan(r, concourseRoot, nil)

			assert.NoError(t, err)

			assert.Len(t, p, 2)
			assert.Equal(t, "cf login -a a -u d -p ******** -o b -s c", p[0].String())
			assert.Equal(t, "cf yay", p[1].String())
		})



	})
}
