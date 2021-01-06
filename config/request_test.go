package config

import (
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
		assert.Error(t, source.Verify())
	}

	validSource := Source{
		API:      "a",
		Org:      "a",
		Space:    "a",
		Username: "a",
		Password: "c",
	}

	assert.Nil(t, validSource.Verify())
}

func TestVerifyErrorsIfNotAllRequiredParamsFieldsAreFilledOut(t *testing.T) {
	missingCommand := Params{
		Command: "",
	}
	assert.Equal(t, ParamsMissingError("command"), missingCommand.Verify())

	missingManifestPath := Params{
		Command: "Something",
	}
	assert.Equal(t, ParamsMissingError("manifestPath"), missingManifestPath.Verify())

}

func TestVerifyErrorsIfNotAllRequiredParamsFieldsForPushFilledOut(t *testing.T) {
	missingTestDomain := Params{
		Command:      PUSH,
		CliVersion:   "cf6",
		ManifestPath: "path",
		TestDomain:   "",
	}
	assert.Equal(t, ParamsMissingError("testDomain"), missingTestDomain.Verify())

	missingAppPath := Params{
		Command:      PUSH,
		CliVersion:   "cf7",
		ManifestPath: "path",
		TestDomain:   "test.com",
		AppPath:      "",
	}
	assert.Equal(t, ParamsMissingError("appPath"), missingAppPath.Verify())

	missingGitRefPath := Params{
		Command:      PUSH,
		CliVersion:   "cf6",
		ManifestPath: "path",
		TestDomain:   "test.com",
		AppPath:      "path",
		GitRefPath:   "",
	}
	assert.Equal(t, ParamsMissingError("gitRefPath"), missingGitRefPath.Verify())

	allesOk := Params{
		Command:      PUSH,
		CliVersion:   "cf6",
		ManifestPath: "path",
		TestDomain:   "test.com",
		AppPath:      "path",
		GitRefPath:   "path",
	}
	assert.Nil(t, allesOk.Verify())
}

func TestVerifyErrorsIfNotAllRequiredParamsFieldsForPromoteFilledOut(t *testing.T) {
	missingTestDomain := Params{
		Command:      PROMOTE,
		CliVersion:   "cf6",
		ManifestPath: "path",
		TestDomain:   "",
	}
	assert.Equal(t, ParamsMissingError("testDomain"), missingTestDomain.Verify())

	allesOk := Params{
		Command:      PROMOTE,
		CliVersion:   "cf6",
		ManifestPath: "path",
		TestDomain:   "test.com",
	}
	assert.Nil(t, allesOk.Verify())
}

func TestVerifyErrorsIfNotAllRequiredParamsFieldsForCleanupFilledOut(t *testing.T) {
	allesOk := Params{
		Command:      CLEANUP,
		CliVersion:   "cf6",
		ManifestPath: "path",
	}
	assert.Nil(t, allesOk.Verify())
}

func TestPreStartCommandForPush(t *testing.T) {
	t.Run("Invalid preStartCommand", func(t *testing.T) {
		invalidParams := Params{
			Command:         PUSH,
			CliVersion:      "cf6",
			ManifestPath:    "path",
			TestDomain:      "test.com",
			AppPath:         "path",
			GitRefPath:      "path",
			PreStartCommand: "something bad",
		}

		expectedError := PreStartCommandError("something bad")

		assert.Equal(t, expectedError, invalidParams.Verify())
	})

	t.Run("Valid preStartCommand", func(t *testing.T) {
		allesOk := Params{
			Command:         PUSH,
			CliVersion:      "cf6",
			ManifestPath:    "path",
			TestDomain:      "test.com",
			AppPath:         "path",
			GitRefPath:      "path",
			PreStartCommand: "cf something good",
		}

		assert.NoError(t, allesOk.Verify())
	})
}

func TestVerifyItDoesntErrorIfAppPathIsEmptyButDockerSpecified(t *testing.T) {
	allesOk := Params{
		Command:        PUSH,
		CliVersion:     "cf6",
		ManifestPath:   "path",
		TestDomain:     "test.com",
		GitRefPath:     "path",
		DockerUsername: "asd",
		DockerPassword: "asd",
	}
	assert.Nil(t, allesOk.Verify())
}
