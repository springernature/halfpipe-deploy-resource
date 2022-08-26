package plan

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"github.com/cloudfoundry-community/go-cfclient"
	"strings"
)

type CleanupPlan interface {
	Plan(manifest manifestparser.Application, summary []cfclient.AppSummary) (pl Plan)
}

type cleanupPlan struct {
}

func (p cleanupPlan) Plan(manifest manifestparser.Application, summary []cfclient.AppSummary) (pl Plan) {
	for _, app := range summary {
		if strings.HasPrefix(app.Name, createDeleteName(manifest.Name, 0)) {
			pl = append(pl, NewCfCommand("delete", app.Name, "-f"))
		}
	}
	return
}

func NewCleanupPlan() CleanupPlan {
	return cleanupPlan{}
}
