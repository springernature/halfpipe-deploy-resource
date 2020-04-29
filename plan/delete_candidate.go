package plan

import (
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"strings"
)

type DeleteCandidatePlan interface {
	Plan(manifest manifest.Application, summary []cfclient.AppSummary) (pl Plan)
}

type deleteCandidatePlan struct {
}

func (p deleteCandidatePlan) Plan(manifest manifest.Application, summary []cfclient.AppSummary) (pl Plan) {
	for _, app := range summary {
		if strings.HasPrefix(app.Name, createDeleteName(manifest.Name, 0)) {
			pl = append(pl, NewCfCommand("delete", app.Name, "-f"))
		}
	}
	return
}

func NewDeleteCandidatePlan() DeleteCandidatePlan {
	return deleteCandidatePlan{}
}
