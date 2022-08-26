package plan

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"errors"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
)

var validRequest = config.Request{
	Source: config.Source{
		API:      "a",
		Org:      "b",
		Space:    "c",
		Username: "d",
		Password: "e",
	},
	Params: config.Params{
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

func (m *ManifestReadWriteStub) ReadManifestNew(path string) (manifestparser.Manifest, error) {
	panic("Should not be used in the test")
}

func (m *ManifestReadWriteStub) WriteManifest(path string, application manifest.Application) error {
	m.writePath = path
	m.savedManifest = manifest.Manifest{
		Applications: []manifest.Application{application},
	}

	return m.writeError
}

func (m ManifestReadWriteStub) WriteManifestNew(path string, manifest manifestparser.Manifest) error {
	panic("Should not be used in the test")
}

func TestErrorsReadingAppManifest(t *testing.T) {
	expectedErr := errors.New("blurgh")
	manifestReader := ManifestReadWriteStub{readError: expectedErr}

	planner := NewPlanner(&manifestReader, nil, nil, nil, nil, nil, nil)

	_, err := planner.Plan(validRequest, nil)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, validRequest.Params.ManifestPath, manifestReader.readPath)
}

func TestErrorsWhenSavingManifestWithUpdatedVars(t *testing.T) {
	expectedErr := errors.New("blurgh")
	manifestReader := ManifestReadWriteStub{
		manifest: manifest.Manifest{
			Applications: []manifest.Application{
				{},
			},
		},
		writeError: expectedErr,
	}

	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	fs.WriteFile(validRequest.Params.GitRefPath, []byte(""), 0777)
	fs.WriteFile(validRequest.Params.BuildVersionPath, []byte(""), 0777)

	planner := NewPlanner(&manifestReader, nil, nil, nil, nil, nil, nil)

	_, err := planner.Plan(validRequest, nil)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, validRequest.Params.ManifestPath, manifestReader.readPath)
	assert.Equal(t, validRequest.Params.ManifestPath, manifestReader.writePath)
}

type fakePushPlanner struct {
	plan      Plan
	dockerTag string
}

type fakeRollingDeployPlanner struct {
	plan      Plan
	dockerTag string
}

type fakeCheckPlanner struct {
	plan Plan
}

type fakePromotePlanner struct {
	plan Plan
}

type fakeCleanupPlanner struct {
	plan Plan
}

type fakeDeleteCandidatePlanner struct {
	plan Plan
}

func (f fakeCheckPlanner) Plan(manifest manifest.Application, org, space string) (pl Plan) {
	return f.plan
}

func (f fakePromotePlanner) Plan(manifest manifest.Application, request config.Request, summary []cfclient.AppSummary) (pl Plan) {
	return f.plan
}

func (f fakeCleanupPlanner) Plan(manifest manifest.Application, summary []cfclient.AppSummary) (pl Plan) {
	return f.plan
}

func (f fakeDeleteCandidatePlanner) Plan(manifest manifest.Application, summary []cfclient.AppSummary) (pl Plan) {
	return f.plan
}

func (f *fakePushPlanner) Plan(manifest manifest.Application, request config.Request) (pl Plan) {
	return f.plan
}

func (f *fakeRollingDeployPlanner) Plan(manifest manifest.Application, request config.Request) (pl Plan) {
	return f.plan
}

func TestCallsOutToCorrectPlanner(t *testing.T) {
	t.Run("Push planner", func(t *testing.T) {
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

		planner := NewPlanner(&manifestReader, &fakePushPlanner{
			plan: Plan{
				NewCfCommand("yay"),
			},
		}, nil, nil, nil, nil, nil)

		r := validRequest
		r.Params.BuildVersionPath = ""
		r.Params.GitRefPath = ""
		p, err := planner.Plan(r, nil)

		assert.NoError(t, err)

		assert.Len(t, p, 3)
		assert.Equal(t, "cf --version", p[0].String())
		assert.Equal(t, "cf login -a a -u d -p ******** -o b -s c", p[1].String())
		assert.Equal(t, "cf yay", p[2].String())
		assert.Equal(t, validRequest.Params.ManifestPath, manifestReader.readPath)
		assert.Equal(t, validRequest.Params.ManifestPath, manifestReader.writePath)
		assert.Equal(t, expectedManifest, manifestReader.savedManifest)
	})

	t.Run("Rolling deploy planner", func(t *testing.T) {

		expectedPath := validRequest.Params.ManifestPath

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

		planner := NewPlanner(&manifestReader, nil, nil, nil, nil, &fakeRollingDeployPlanner{
			plan: Plan{
				NewCfCommand("yay"),
			},
		}, nil)

		r := validRequest
		r.Params.Command = config.ROLLING_DEPLOY
		r.Params.BuildVersionPath = ""
		r.Params.GitRefPath = ""
		p, err := planner.Plan(r, nil)

		assert.NoError(t, err)

		assert.Len(t, p, 3)
		assert.Equal(t, "cf --version", p[0].String())
		assert.Equal(t, "cf login -a a -u d -p ******** -o b -s c", p[1].String())
		assert.Equal(t, "cf yay", p[2].String())
		assert.Equal(t, expectedPath, manifestReader.readPath)
		assert.Equal(t, expectedPath, manifestReader.writePath)
		assert.Equal(t, expectedManifest, manifestReader.savedManifest)
	})

	t.Run("Check planner", func(t *testing.T) {
		manifestReader := ManifestReadWriteStub{
			manifest: manifest.Manifest{
				Applications: []manifest.Application{
					{},
				},
			},
		}

		planner := NewPlanner(&manifestReader, nil, fakeCheckPlanner{
			plan: Plan{
				NewCfCommand("yay"),
			},
		}, nil, nil, nil, nil)

		r := validRequest
		r.Params.Command = config.CHECK

		p, err := planner.Plan(r, nil)

		assert.NoError(t, err)

		assert.Len(t, p, 1)
		assert.Equal(t, "cf yay", p[0].String())
	})

	t.Run("Promote planner", func(t *testing.T) {
		manifestReader := ManifestReadWriteStub{
			manifest: manifest.Manifest{
				Applications: []manifest.Application{
					{},
				},
			},
		}

		planner := NewPlanner(&manifestReader, nil, nil, fakePromotePlanner{
			plan: Plan{
				NewCfCommand("yay"),
			},
		}, nil, nil, nil)

		r := validRequest
		r.Params.Command = config.PROMOTE

		p, err := planner.Plan(r, nil)

		assert.NoError(t, err)

		assert.Len(t, p, 3)
		assert.Equal(t, "cf --version", p[0].String())
		assert.Equal(t, "cf login -a a -u d -p ******** -o b -s c", p[1].String())
		assert.Equal(t, "cf yay", p[2].String())
	})

	t.Run("Cleanup planner", func(t *testing.T) {
		manifestReader := ManifestReadWriteStub{
			manifest: manifest.Manifest{
				Applications: []manifest.Application{
					{},
				},
			},
		}

		planner := NewPlanner(&manifestReader, nil, nil, nil, fakeCleanupPlanner{
			plan: Plan{
				NewCfCommand("yay"),
			},
		}, nil, nil)

		t.Run("Works with cleanup command", func(t *testing.T) {
			r := validRequest
			r.Params.Command = config.CLEANUP

			p, err := planner.Plan(r, nil)

			assert.NoError(t, err)

			assert.Len(t, p, 3)
			assert.Equal(t, "cf --version", p[0].String())
			assert.Equal(t, "cf login -a a -u d -p ******** -o b -s c", p[1].String())
			assert.Equal(t, "cf yay", p[2].String())
		})

		t.Run("Works with a delete command", func(t *testing.T) {
			r := validRequest
			r.Params.Command = config.DELETE

			p, err := planner.Plan(r, nil)

			assert.NoError(t, err)

			assert.Len(t, p, 3)
			assert.Equal(t, "cf --version", p[0].String())
			assert.Equal(t, "cf login -a a -u d -p ******** -o b -s c", p[1].String())
			assert.Equal(t, "cf yay", p[2].String())
		})

	})

	t.Run("Delete Candidate planner", func(t *testing.T) {
		manifestReader := ManifestReadWriteStub{
			manifest: manifest.Manifest{
				Applications: []manifest.Application{
					{},
				},
			},
		}

		planner := NewPlanner(&manifestReader, nil, nil, nil, nil, nil, &fakeDeleteCandidatePlanner{
			plan: Plan{
				NewCfCommand("yay"),
			},
		})

		r := validRequest
		r.Params.Command = config.DELETE_CANDIDATE

		p, err := planner.Plan(r, nil)

		assert.NoError(t, err)

		assert.Len(t, p, 3)
		assert.Equal(t, "cf --version", p[0].String())
		assert.Equal(t, "cf login -a a -u d -p ******** -o b -s c", p[1].String())
		assert.Equal(t, "cf yay", p[2].String())
	})
}
