package plan

import (
	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVerifyErrorsIfNotAllSourceFieldsAreFilledOut(t *testing.T) {
	invalidSourceRequests := []Source{
		{
			API:      "",
			Org:      "",
			Space:    "",
			Username: "",
			Password: "",
		},

		{
			API:      "a",
			Org:      "",
			Space:    "",
			Username: "",
			Password: "",
		},

		{
			API:      "a",
			Org:      "a",
			Space:    "",
			Username: "",
			Password: "",
		},

		{
			API:      "a",
			Org:      "a",
			Space:    "a",
			Username: "",
			Password: "",
		},

		{
			API:      "a",
			Org:      "a",
			Space:    "a",
			Username: "a",
			Password: "",
		},
	}

	for _, source := range invalidSourceRequests {
		assert.Error(t, VerifyRequestSource(source))
	}

	validSource := Source{
		API:      "a",
		Org:      "a",
		Space:    "a",
		Username: "a",
		Password: "c",
	}

	assert.Nil(t, VerifyRequestSource(validSource))
}

func TestVerifyErrorsIfNotAllRequiredParamsFieldsAreFilledOut(t *testing.T) {
	missingCommand := Params{
		Command: "",
	}
	assert.Equal(t, ParamsMissingError("command"), VerifyRequestParams(missingCommand))

	missingManifestPath := Params{
		Command: "Something",
	}
	assert.Equal(t, ParamsMissingError("manifestPath"), VerifyRequestParams(missingManifestPath))

}

func TestVerifyErrorsIfNotAllRequiredParamsFieldsForPushFilledOut(t *testing.T) {
	missingTestDomain := Params{
		Command:      config.PUSH,
		ManifestPath: "path",
		TestDomain:   "",
	}
	assert.Equal(t, ParamsMissingError("testDomain"), VerifyRequestParams(missingTestDomain))

	missingAppPath := Params{
		Command:      config.PUSH,
		ManifestPath: "path",
		TestDomain:   "test.com",
		AppPath:      "",
	}
	assert.Equal(t, ParamsMissingError("appPath"), VerifyRequestParams(missingAppPath))

	missingGitRefPath := Params{
		Command:      config.PUSH,
		ManifestPath: "path",
		TestDomain:   "test.com",
		AppPath:      "path",
		GitRefPath:   "",
	}
	assert.Equal(t, ParamsMissingError("gitRefPath"), VerifyRequestParams(missingGitRefPath))

	allesOk := Params{
		Command:      config.PUSH,
		ManifestPath: "path",
		TestDomain:   "test.com",
		AppPath:      "path",
		GitRefPath:   "path",
	}
	assert.Nil(t, VerifyRequestParams(allesOk))
}

func TestVerifyErrorsIfNotAllRequiredParamsFieldsForPromoteFilledOut(t *testing.T) {
	missingTestDomain := Params{
		Command:      config.PROMOTE,
		ManifestPath: "path",
		TestDomain:   "",
	}
	assert.Equal(t, ParamsMissingError("testDomain"), VerifyRequestParams(missingTestDomain))

	allesOk := Params{
		Command:      config.PROMOTE,
		ManifestPath: "path",
		TestDomain:   "test.com",
	}
	assert.Nil(t, VerifyRequestParams(allesOk))
}

func TestVerifyErrorsIfNotAllRequiredParamsFieldsForCleanupFilledOut(t *testing.T) {
	allesOk := Params{
		Command:      config.CLEANUP,
		ManifestPath: "path",
	}
	assert.Nil(t, VerifyRequestParams(allesOk))
}

func TestPreStartCommandForPush(t *testing.T) {
	t.Run("Invalid preStartCommand", func(t *testing.T) {
		invalidParams := Params{
			Command:         config.PUSH,
			ManifestPath:    "path",
			TestDomain:      "test.com",
			AppPath:         "path",
			GitRefPath:      "path",
			PreStartCommand: "something bad",
		}

		expectedError := PreStartCommandError("something bad")

		assert.Equal(t, expectedError, VerifyRequestParams(invalidParams))
	})

	t.Run("Valid preStartCommand", func(t *testing.T) {
		allesOk := Params{
			Command:         config.PUSH,
			ManifestPath:    "path",
			TestDomain:      "test.com",
			AppPath:         "path",
			GitRefPath:      "path",
			PreStartCommand: "cf something good",
		}

		assert.NoError(t, VerifyRequestParams(allesOk))
	})
}

func TestRollingCommands(t *testing.T) {
	t.Run("DEPLOY_ROLLING", func(t *testing.T) {
		t.Run("Missing params", func(t *testing.T) {
			missingCommand := Params{
				Command: config.DEPLOY_ROLLING,
				ManifestPath: "something",
			}
			assert.Equal(t, ParamsMissingError("appPath"), VerifyRequestParams(missingCommand))
		})

		t.Run("All required params", func(t *testing.T) {
			complete := Params{
				Command: config.DEPLOY_ROLLING,
				ManifestPath: "something",
				TestDomain: "blah",
				AppPath: "blah",
				GitRefPath: "blah",
			}
			assert.NoError(t, VerifyRequestParams(complete))

		})
	})

	t.Run("DELETE_TEST", func(t *testing.T) {
		t.Run("Missing params", func(t *testing.T) {
			missingCommand := Params{
				Command: config.DELETE_TEST,
			}
			assert.Equal(t, ParamsMissingError("manifestPath"), VerifyRequestParams(missingCommand))
		})

		t.Run("All required params", func(t *testing.T) {
			complete := Params{
				Command: config.DELETE_TEST,
				ManifestPath: "something",
			}
			assert.NoError(t, VerifyRequestParams(complete))
		})
	})

}

func TestVerifyItDoesntErrorIfAppPathIsEmptyButDockerSpecified(t *testing.T) {
	allesOk := Params{
		Command:        config.PUSH,
		ManifestPath:   "path",
		TestDomain:     "test.com",
		GitRefPath:     "path",
		DockerUsername: "asd",
		DockerPassword: "asd",
	}
	assert.Nil(t, VerifyRequestParams(allesOk))
}
