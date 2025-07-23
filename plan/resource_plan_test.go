package plan

import (
	"errors"
	"testing"

	"code.cloudfoundry.org/cli/util/manifestparser"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/spf13/afero"
	halfpipe_deploy_resource "github.com/springernature/halfpipe-deploy-resource"
	"github.com/stretchr/testify/assert"

	"github.com/springernature/halfpipe-deploy-resource/config"
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
		ManifestPath: "manifest.yml",
		AppPath:      "",
		TestDomain:   "kehe.com",
		Command:      config.PUSH,
		GitRefPath:   "gitRefPath",

		BuildVersionPath: "buildVersionPath",
		Vars: map[string]string{
			"VAR2": "bb",
			"VAR4": "cc",
		},
		Team:   "myTeam",
		GitUri: "git@github.com:springernature/halfpipe-deploy-resource.git",
	},
}

type ManifestReadWriteStub struct {
	manifest          manifestparser.Manifest
	manifestReadError error

	savedManifest     manifestparser.Manifest
	saveManifestError error

	readPath  string
	writePath string
}

func (m *ManifestReadWriteStub) ReadManifest(path string) (manifestparser.Manifest, error) {
	m.readPath = path
	return m.manifest, m.manifestReadError
}

func (m *ManifestReadWriteStub) WriteManifest(path string, manifest manifestparser.Manifest) error {
	m.writePath = path
	m.savedManifest = manifest

	return m.saveManifestError
}

func TestErrorsReadingAppManifest(t *testing.T) {
	expectedErr := errors.New("blurgh")
	manifestReader := ManifestReadWriteStub{manifestReadError: expectedErr}

	planner := NewPlanner(&manifestReader, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := planner.Plan(validRequest, nil)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, validRequest.Params.ManifestPath, manifestReader.readPath)
}

func TestErrorsWhenSavingManifestWithUpdatedVars(t *testing.T) {
	expectedErr := errors.New("blurgh")
	manifestReader := ManifestReadWriteStub{
		manifest: halfpipe_deploy_resource.ParseManifest(`applications:
- name: MyApp`),
		saveManifestError: expectedErr,
	}

	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	fs.WriteFile(validRequest.Params.GitRefPath, []byte(""), 0777)
	fs.WriteFile(validRequest.Params.BuildVersionPath, []byte(""), 0777)

	planner := NewPlanner(&manifestReader, nil, nil, nil, nil, nil, nil, nil, nil, nil)

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

func (f fakeCheckPlanner) Plan(manifest manifestparser.Application, org, space string) (pl Plan) {
	return f.plan
}

func (f fakePromotePlanner) Plan(manifest manifestparser.Application, request config.Request, summary []cfclient.AppSummary) (pl Plan) {
	return f.plan
}

func (f fakeCleanupPlanner) Plan(manifest manifestparser.Application, summary []cfclient.AppSummary) (pl Plan) {
	return f.plan
}

func (f fakeDeleteCandidatePlanner) Plan(manifest manifestparser.Application, summary []cfclient.AppSummary) (pl Plan) {
	return f.plan
}

func (f *fakePushPlanner) Plan(manifest manifestparser.Application, request config.Request) (pl Plan) {
	return f.plan
}

func (f *fakeRollingDeployPlanner) Plan(manifest manifestparser.Application, request config.Request) (pl Plan) {
	return f.plan
}

func TestCallsOutToCorrectPlanner(t *testing.T) {
	t.Run("Push planner", func(tt *testing.T) {
		tt.Run("Pushes", func(ttt *testing.T) {
			expectedManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp
  env:
    VAR2: bb
    VAR4: cc
    OTEL_EXPORTER_OTLP_PROTOCOL: "http/protobuf"
    OTEL_EXPORTER_OTLP_HEADERS: "X-Scope-OrgId=myTeam"
    OTEL_SERVICE_NAME: "myApp"
    OTEL_EXPORTER_OTLP_ENDPOINT: "http://opentelemetry-sink.tracing.springernature.io:80"
    OTEL_PROPAGATORS: "tracecontext"
    OTEL_RESOURCE_ATTRIBUTES: "service.namespace=b/c,job=b/c/myApp,cloudfoundry.app.name=myApp,cloudfoundry.app.org.name=b,cloudfoundry.app.space.name=c"
  metadata:
    labels:
      team: myTeam
      gitRepo: halfpipe-deploy-resource
`)

			manifestReader := ManifestReadWriteStub{
				manifest: halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp`),
			}

			planner := NewPlanner(&manifestReader, &fakePushPlanner{
				plan: Plan{
					NewCfCommand("yay"),
				},
			}, nil, nil, nil, nil, nil, nil, NewCheckLabelsPlan(), nil)

			r := validRequest
			r.Params.BuildVersionPath = ""
			r.Params.GitRefPath = ""
			p, err := planner.Plan(r, nil)

			assert.NoError(ttt, err)

			assert.Len(ttt, p, 4)
			assert.Equal(ttt, "cf --version", p[0].String())
			assert.Equal(ttt, "cf login -a a -u d -p ******** -o b -s c", p[1].String())
			assert.Equal(ttt, "Linting application", p[2].String())
			assert.Equal(ttt, "cf yay", p[3].String())
			assert.Equal(ttt, validRequest.Params.ManifestPath, manifestReader.readPath)
			assert.Equal(ttt, validRequest.Params.ManifestPath, manifestReader.writePath)
			assert.Equal(ttt, expectedManifest, manifestReader.savedManifest)
		})

		tt.Run("Changes team label if present", func(ttt *testing.T) {
			expectedManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp
  env:
    VAR2: bb
    VAR4: cc
    OTEL_EXPORTER_OTLP_PROTOCOL: "http/protobuf"
    OTEL_EXPORTER_OTLP_HEADERS: "X-Scope-OrgId=myTeam"
    OTEL_SERVICE_NAME: "myApp"
    OTEL_EXPORTER_OTLP_ENDPOINT: "http://opentelemetry-sink.tracing.springernature.io:80"
    OTEL_PROPAGATORS: "tracecontext"
    OTEL_RESOURCE_ATTRIBUTES: "service.namespace=b/c,job=b/c/myApp,cloudfoundry.app.name=myApp,cloudfoundry.app.org.name=b,cloudfoundry.app.space.name=c"
  metadata:
    annotations:
      someAnnotation: yo
    labels:
      myLabel: myValue
      environment: dev
      team: myTeam
      gitRepo: halfpipe-deploy-resource
`)
			manifestReader := ManifestReadWriteStub{
				manifest: halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp
  metadata:
    annotations:
      someAnnotation: yo
    labels:
      myLabel: myValue
      environment: dev
      team: myHardcodedTeam
`),
			}

			planner := NewPlanner(&manifestReader, &fakePushPlanner{
				plan: Plan{
					NewCfCommand("yay"),
				},
			}, nil, nil, nil, nil, nil, nil, NewCheckLabelsPlan(), nil)

			r := validRequest
			r.Params.BuildVersionPath = ""
			r.Params.GitRefPath = ""
			_, err := planner.Plan(r, nil)

			assert.NoError(ttt, err)
			assert.Equal(ttt, expectedManifest, manifestReader.savedManifest)
		})

		tt.Run("Does nothing with labels if team is not defined and gitUri unset", func(ttt *testing.T) {
			expectedManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp
  env:
    VAR2: bb
    VAR4: cc
    OTEL_EXPORTER_OTLP_PROTOCOL: "http/protobuf"
    OTEL_EXPORTER_OTLP_HEADERS: "X-Scope-OrgId=anonymous"
    OTEL_SERVICE_NAME: "myApp"
    OTEL_EXPORTER_OTLP_ENDPOINT: "http://opentelemetry-sink.tracing.springernature.io:80"
    OTEL_PROPAGATORS: "tracecontext"
    OTEL_RESOURCE_ATTRIBUTES: "service.namespace=b/c,job=b/c/myApp,cloudfoundry.app.name=myApp,cloudfoundry.app.org.name=b,cloudfoundry.app.space.name=c"
`)
			manifestReader := ManifestReadWriteStub{
				manifest: halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp
`),
			}

			planner := NewPlanner(&manifestReader, &fakePushPlanner{
				plan: Plan{
					NewCfCommand("yay"),
				},
			}, nil, nil, nil, nil, nil, nil, NewCheckLabelsPlan(), nil)

			r := validRequest
			r.Params.BuildVersionPath = ""
			r.Params.GitRefPath = ""
			r.Params.GitUri = ""
			r.Params.Team = ""
			_, err := planner.Plan(r, nil)

			assert.NoError(ttt, err)

			assert.Equal(ttt, expectedManifest, manifestReader.savedManifest)
		})

		tt.Run("Does nothing with labels if team is not defined but there are some annotations", func(ttt *testing.T) {
			expectedManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp
  env:
    VAR2: bb
    VAR4: cc
    OTEL_EXPORTER_OTLP_PROTOCOL: "http/protobuf"
    OTEL_EXPORTER_OTLP_HEADERS: "X-Scope-OrgId=anonymous"
    OTEL_SERVICE_NAME: "myApp"
    OTEL_EXPORTER_OTLP_ENDPOINT: "http://opentelemetry-sink.tracing.springernature.io:80" 
    OTEL_PROPAGATORS: "tracecontext"
    OTEL_RESOURCE_ATTRIBUTES: "service.namespace=b/c,job=b/c/myApp,cloudfoundry.app.name=myApp,cloudfoundry.app.org.name=b,cloudfoundry.app.space.name=c"
  metadata:
    annotations:
     a: b
`)
			manifestReader := ManifestReadWriteStub{
				manifest: halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp
  metadata:
    annotations:
      a: b
`),
			}

			planner := NewPlanner(&manifestReader, &fakePushPlanner{
				plan: Plan{
					NewCfCommand("yay"),
				},
			}, nil, nil, nil, nil, nil, nil, NewCheckLabelsPlan(), nil)

			r := validRequest
			r.Params.BuildVersionPath = ""
			r.Params.GitRefPath = ""
			r.Params.GitUri = ""
			r.Params.Team = ""
			_, err := planner.Plan(r, nil)

			assert.NoError(ttt, err)

			assert.Equal(ttt, expectedManifest, manifestReader.savedManifest)
		})

		tt.Run("Does not overwrite OTEL stuff thats already in the manifest", func(ttt *testing.T) {
			expectedManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp
  env:
    VAR2: bb
    VAR4: cc
    OTEL_EXPORTER_OTLP_PROTOCOL: "http/protobuf"
    OTEL_EXPORTER_OTLP_HEADERS: "X-Scope-OrgId=anonymous"
    OTEL_SERVICE_NAME: "BLAAAAH"
    OTEL_EXPORTER_OTLP_ENDPOINT: "http://opentelemetry-sink.tracing.springernature.io:80"
    OTEL_PROPAGATORS: "tracecontext"
    OTEL_RESOURCE_ATTRIBUTES: "service.namespace=b/c,job=b/c/myApp,cloudfoundry.app.name=myApp,cloudfoundry.app.org.name=b,cloudfoundry.app.space.name=c"
  metadata:
    annotations:
     a: b
`)
			manifestReader := ManifestReadWriteStub{
				manifest: halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp
  env:
    OTEL_SERVICE_NAME: "BLAAAAH"
  metadata:
    annotations:
      a: b
`),
			}

			planner := NewPlanner(&manifestReader, &fakePushPlanner{
				plan: Plan{
					NewCfCommand("yay"),
				},
			}, nil, nil, nil, nil, nil, nil, NewCheckLabelsPlan(), nil)

			r := validRequest
			r.Params.BuildVersionPath = ""
			r.Params.GitRefPath = ""
			r.Params.GitUri = ""
			r.Params.Team = ""
			_, err := planner.Plan(r, nil)

			assert.NoError(ttt, err)

			assert.Equal(ttt, expectedManifest, manifestReader.savedManifest)
		})
	})

	t.Run("Rolling deploy planner", func(tt *testing.T) {
		expectedPath := validRequest.Params.ManifestPath

		expectedManifest := halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp
  env:
    VAR2: bb
    VAR4: cc
    OTEL_EXPORTER_OTLP_PROTOCOL: "http/protobuf"
    OTEL_EXPORTER_OTLP_HEADERS: "X-Scope-OrgId=myTeam"
    OTEL_SERVICE_NAME: "myApp"
    OTEL_EXPORTER_OTLP_ENDPOINT: "http://opentelemetry-sink.tracing.springernature.io:80"
    OTEL_PROPAGATORS: "tracecontext"
    OTEL_RESOURCE_ATTRIBUTES: "service.namespace=b/c,job=b/c/myApp,cloudfoundry.app.name=myApp,cloudfoundry.app.org.name=b,cloudfoundry.app.space.name=c"
  metadata:
    labels:
      team: myTeam
      gitRepo: halfpipe-deploy-resource
`)

		manifestReader := ManifestReadWriteStub{
			manifest: halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp`),
		}

		planner := NewPlanner(&manifestReader, nil, nil, nil, nil, &fakeRollingDeployPlanner{
			plan: Plan{
				NewCfCommand("yay"),
			},
		}, nil, nil, NewCheckLabelsPlan(), nil)

		r := validRequest
		r.Params.Command = config.ROLLING_DEPLOY
		r.Params.BuildVersionPath = ""
		r.Params.GitRefPath = ""
		p, err := planner.Plan(r, nil)

		assert.NoError(tt, err)

		assert.Len(tt, p, 4)
		assert.Equal(tt, "cf --version", p[0].String())
		assert.Equal(tt, "cf login -a a -u d -p ******** -o b -s c", p[1].String())
		assert.Equal(tt, "Linting application", p[2].String())
		assert.Equal(tt, "cf yay", p[3].String())
		assert.Equal(tt, expectedPath, manifestReader.readPath)
		assert.Equal(tt, expectedPath, manifestReader.writePath)
		assert.Equal(tt, expectedManifest, manifestReader.savedManifest)
	})

	t.Run("Check planner", func(t *testing.T) {
		manifestReader := ManifestReadWriteStub{
			manifest: halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp`),
		}

		planner := NewPlanner(&manifestReader, nil, fakeCheckPlanner{
			plan: Plan{
				NewCfCommand("yay"),
			},
		}, nil, nil, nil, nil, nil, nil, nil)

		r := validRequest
		r.Params.Command = config.CHECK

		p, err := planner.Plan(r, nil)

		assert.NoError(t, err)

		assert.Len(t, p, 1)
		assert.Equal(t, "cf yay", p[0].String())
	})

	t.Run("Promote planner", func(t *testing.T) {
		manifestReader := ManifestReadWriteStub{
			manifest: halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp`),
		}

		planner := NewPlanner(&manifestReader, nil, nil, fakePromotePlanner{
			plan: Plan{
				NewCfCommand("yay"),
			},
		}, nil, nil, nil, nil, nil, nil)

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
			manifest: halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp`),
		}

		planner := NewPlanner(&manifestReader, nil, nil, nil, fakeCleanupPlanner{
			plan: Plan{
				NewCfCommand("yay"),
			},
		}, nil, nil, nil, nil, nil)

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
			manifest: halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp`),
		}

		planner := NewPlanner(&manifestReader, nil, nil, nil, nil, nil, &fakeDeleteCandidatePlanner{
			plan: Plan{
				NewCfCommand("yay"),
			},
		}, nil, nil, nil)

		r := validRequest
		r.Params.Command = config.DELETE_CANDIDATE

		p, err := planner.Plan(r, nil)

		assert.NoError(t, err)

		assert.Len(t, p, 3)
		assert.Equal(t, "cf --version", p[0].String())
		assert.Equal(t, "cf login -a a -u d -p ******** -o b -s c", p[1].String())
		assert.Equal(t, "cf yay", p[2].String())
	})

	t.Run("Logs planner", func(t *testing.T) {
		manifestReader := ManifestReadWriteStub{
			manifest: halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp`),
		}

		planner := NewPlanner(&manifestReader, nil, nil, nil, nil, nil, nil, NewLogsPlan(), nil, nil)

		r := validRequest
		r.Params.Command = config.LOGS

		p, err := planner.Plan(r, nil)

		assert.NoError(t, err)

		assert.Len(t, p, 3)
		assert.Equal(t, "cf --version", p[0].String())
		assert.Equal(t, "cf login -a a -u d -p ******** -o b -s c", p[1].String())
		assert.Equal(t, "cf logs myApp-CANDIDATE --recent", p[2].String())
	})

	t.Run("SSO planner", func(t *testing.T) {
		manifestReader := ManifestReadWriteStub{
			manifest: halfpipe_deploy_resource.ParseManifest(`applications:
- name: myApp`),
		}

		planner := NewPlanner(&manifestReader, nil, nil, nil, nil, nil, nil, nil, nil, NewSSOPlan())

		r := validRequest
		r.Params.Command = config.SSO
		r.Params.SSOHost = "myHost"

		p, err := planner.Plan(r, nil)

		assert.NoError(t, err)

		assert.Len(t, p, 5)
		assert.Equal(t, "cf --version", p[0].String())
		assert.Equal(t, "cf login -a a -u d -p ******** -o b -s c", p[1].String())
		assert.Equal(t, "cf service sso || cf create-user-provided-service sso -r https://ee-sso.public.springernature.app", p[2].String())
		assert.Equal(t, "cf route public.springernature.app -n myHost || cf create-route public.springernature.app -n myHost", p[3].String())
		assert.Equal(t, "cf bind-route-service public.springernature.app -n myHost sso", p[4].String())
	})
}
