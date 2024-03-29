package plan

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"github.com/cloudfoundry-community/go-cfclient"
)

type DeleteCandidatePlan interface {
	Plan(manifest manifestparser.Application, summary []cfclient.AppSummary) (pl Plan)
}

type deleteCandidatePlan struct {
}

func (p deleteCandidatePlan) Plan(manifest manifestparser.Application, summary []cfclient.AppSummary) (pl Plan) {
	pl = Plan{}

	for _, appSummary := range summary {
		if appSummary.Name == createCandidateAppName(manifest.Name) {
			pl = append(pl, NewCfCommand("delete", createCandidateAppName(manifest.Name), "-f"))
		}
	}
	return
}

func NewDeleteCandidatePlan() DeleteCandidatePlan {
	return deleteCandidatePlan{}
}
