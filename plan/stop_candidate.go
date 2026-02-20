package plan

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"github.com/cloudfoundry-community/go-cfclient"
)

type StopCandidatePlan interface {
	Plan(manifest manifestparser.Application, summary []cfclient.AppSummary) (pl Plan)
}

type stopCandidatePlan struct {
}

func (p stopCandidatePlan) Plan(manifest manifestparser.Application, summary []cfclient.AppSummary) (pl Plan) {
	pl = Plan{}

	for _, appSummary := range summary {
		if appSummary.Name == createCandidateAppName(manifest.Name) {
			pl = append(pl, NewCfCommand("stop", createCandidateAppName(manifest.Name)))
		}
	}
	return
}

func NewStopCandidatePlan() StopCandidatePlan {
	return stopCandidatePlan{}
}
