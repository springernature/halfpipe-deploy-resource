package plan

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/springernature/halfpipe-deploy-resource/config"
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