package config

import (
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestIsAction(t *testing.T) {
	t.Run("when github workspace is set", func(t *testing.T) {
		rr := RequestReader{
			environ: map[string]string{
				"GITHUB_WORKSPACE": "/github/workspace",
			},
		}

		assert.True(t, rr.isActions())
	})
	t.Run("when github workspace is not set", func(t *testing.T) {
		rr := RequestReader{}

		assert.False(t, rr.isActions())
	})
}

func TestReadRequest(t *testing.T) {
	t.Run("Action", func(t *testing.T) {
		env := map[string]string{
			"INPUT_API":          "api",
			"INPUT_ORG":          "org",
			"INPUT_SPACE":        "space",
			"INPUT_USERNAME":     "username",
			"INPUT_PASSWORD":     "password",
			"INPUT_COMMAND":      "command",
			"INPUT_MANIFESTPATH": "app/cf/manifest.yml",
			"INPUT_APPPATH":      "app",
			"INPUT_TESTDOMAIN":   "test domain",
			"INPUT_DOCKERTAG":    "docker-tag",
			"GITHUB_SHA":         "ref",
			"GITHUB_RUN_NUMBER":  "run number",
			"GITHUB_WORKSPACE":   "/github/workspace",
			"CF_ENV_VAR_VAR":     "a",
			"CF_ENV_VAR_VAR2":    "b",
			"CF_ENV_VAR_var_3":   "c",
		}

		expected := Request{
			Source: Source{
				API:      "api",
				Org:      "org",
				Space:    "space",
				Username: "username",
				Password: "password",
			},
			Params: Params{
				Command:      "command",
				ManifestPath: "/github/workspace/app/cf/manifest.yml",
				AppPath:      "/github/workspace/app",
				TestDomain:   "test domain",
				DockerTag:    "docker-tag",
				CliVersion:   "cf6",
				Vars: map[string]string{
					"VAR":   "a",
					"VAR2":  "b",
					"var_3": "c",
				},
			},
			Metadata: Metadata{
				GitRef:    "ref",
				Version:   "run number",
				DockerTag: "docker-tag",
				IsActions: true,
			},
		}

		rr := NewRequestReader([]string{}, env, nil, afero.Afero{})
		req, err := rr.ReadRequest()

		assert.NoError(t, err)
		assert.Equal(t, expected, req)
	})

	t.Run("Concourse", func(t *testing.T) {
		validRequestWithoutVersionPath := `{
   "source": {
      "api":"api",
      "org":"org",
      "password":"password",
      "space":"space",
      "username":"username"
   },
   "params": {
      "appPath":"git/app",
      "command":"halfpipe-push",
      "gitRefPath":"git/.git/ref",
      "manifestPath":"git/app/cf/manifest-qa.yml",
      "testDomain":"springernature.app"
   }
}`
		validRequestWithVersionPath := `{
   "source": {
      "api":"api",
      "org":"org",
      "password":"password",
      "space":"space",
      "username":"username"
   },
   "params": {
      "appPath":"git/app",
      "buildVersionPath":"version/version",
      "command":"halfpipe-push",
      "gitRefPath":"git/.git/ref",
      "manifestPath":"git/app/cf/manifest-qa.yml",
      "testDomain":"springernature.app"
   }
}`
		t.Run("fails to read git ref file", func(t *testing.T) {
			fs := afero.Afero{Fs: afero.NewMemMapFs()}

			stdin := strings.NewReader(validRequestWithoutVersionPath)
			rr := NewRequestReader([]string{"/opt/resource/out", "/tmp/buildDir"}, map[string]string{}, stdin, fs)

			_, err := rr.ReadRequest()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "open /tmp/buildDir/git/.git/ref: file does not exist")
		})

		t.Run("fails to read version file", func(t *testing.T) {
			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			fs.WriteFile("/tmp/buildDir/git/.git/ref", []byte("ref"), 0777)
			stdin := strings.NewReader(validRequestWithVersionPath)
			rr := NewRequestReader([]string{"/opt/resource/out", "/tmp/buildDir"}, map[string]string{}, stdin, fs)

			_, err := rr.ReadRequest()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "open /tmp/buildDir/version/version: file does not exist")
		})

		t.Run("no version", func(t *testing.T) {
			expected := Request{
				Source: Source{
					API:      "api",
					Org:      "org",
					Space:    "space",
					Username: "username",
					Password: "password",
				},
				Params: Params{
					Command:      "halfpipe-push",
					GitRefPath:   "/tmp/buildDir/git/.git/ref",
					ManifestPath: "/tmp/buildDir/git/app/cf/manifest-qa.yml",
					AppPath:      "/tmp/buildDir/git/app",
					TestDomain:   "springernature.app",
					CliVersion:   "cf6",
				},
				Metadata: Metadata{
					GitRef:    "ref",
					IsActions: false,
				},
			}

			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			fs.WriteFile("/tmp/buildDir/git/.git/ref", []byte("ref"), 0777)
			stdin := strings.NewReader(validRequestWithoutVersionPath)
			rr := NewRequestReader([]string{"/opt/resource/out", "/tmp/buildDir"}, map[string]string{}, stdin, fs)

			request, err := rr.ReadRequest()
			assert.NoError(t, err)
			assert.Equal(t, expected, request)

		})

		t.Run("with version", func(t *testing.T) {
			expected := Request{
				Source: Source{
					API:      "api",
					Org:      "org",
					Space:    "space",
					Username: "username",
					Password: "password",
				},
				Params: Params{
					Command:          "halfpipe-push",
					GitRefPath:       "/tmp/buildDir/git/.git/ref",
					BuildVersionPath: "/tmp/buildDir/version/version",
					ManifestPath:     "/tmp/buildDir/git/app/cf/manifest-qa.yml",
					AppPath:          "/tmp/buildDir/git/app",
					TestDomain:       "springernature.app",
					CliVersion:       "cf6",
				},
				Metadata: Metadata{
					GitRef:  "ref",
					Version: "version",
				},
			}

			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			fs.WriteFile("/tmp/buildDir/git/.git/ref", []byte("ref"), 0777)
			fs.WriteFile("/tmp/buildDir/version/version", []byte("version"), 0777)
			stdin := strings.NewReader(validRequestWithVersionPath)
			rr := NewRequestReader([]string{"/opt/resource/out", "/tmp/buildDir"}, map[string]string{}, stdin, fs)

			request, err := rr.ReadRequest()
			assert.NoError(t, err)
			assert.Equal(t, expected, request)

			t.Run("/tmp/request contains something", func(t *testing.T) {
				content, err := fs.ReadFile("/tmp/request")
				assert.NoError(t, err)
				assert.NotEmpty(t, content)
			})
		})
	})
}
