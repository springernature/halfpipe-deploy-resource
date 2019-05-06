package fixes

import (
	"fmt"
	"github.com/springernature/halfpipe-deploy-resource/plan"
	"strings"
)

var suggestDeveloperSpaceRole = func(log []byte, request plan.Request) (err error) {
	if strings.Contains(string(log), "") {
		errorMsg := `'%s' does not have 'SpaceDeveloper' permissions on org/space '%s/%s'
To fix ask your org admin to run 'cf set-space-role %s %s %s SpaceDeveloper'`
		err = fmt.Errorf(errorMsg,
			request.Source.Username,
			request.Source.Org,
			request.Source.Space,
			request.Source.Username,
			request.Source.Org,
			request.Source.Space,
		)
	}
	return
}

func SuggestFix(log []byte, request plan.Request) (fixes []error) {
	if err := suggestDeveloperSpaceRole(log, request); err != nil {
		fixes = append(fixes, err)
	}

	return
}
