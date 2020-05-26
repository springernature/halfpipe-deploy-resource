package plan

import (
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
)

type DeleteCandidatePlan interface {
	Plan(manifest manifest.Application, summary []cfclient.AppSummary) (pl Plan)
}

type deleteCandidatePlan struct {
}

func (p deleteCandidatePlan) Plan(manifest manifest.Application, summary []cfclient.AppSummary) (pl Plan) {
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
