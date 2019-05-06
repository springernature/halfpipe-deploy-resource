package fixes

import (
	"github.com/springernature/halfpipe-deploy-resource/plan"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFixForNotAuthorizedToPerformAction(t *testing.T) {
	r := plan.Request{
		Source: plan.Source{
			Org:      "myOrg",
			Space:    "mySpace",
			Username: "myUser",
		},
	}

	errorLog := []byte(`Getting app info...
Creating app with these attributes...
+ name:         integration-test-app-CANDIDATE
  path:         /Users/simonjohansson/src/halfpipe-deploy-resource/.integration_test
  buildpacks:
+   staticfile_buildpack
  env:
+   GIT_REVISION

Creating app integration-test-app-CANDIDATE...
You are not authorized to perform the requested action
FAILED
exit status 1
`)

	fixes := SuggestFix(errorLog, r)
	assert.Len(t, fixes, 1)
	assert.Contains(t, fixes, suggestDeveloperSpaceRole(errorLog, r))
}
