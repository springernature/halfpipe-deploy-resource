package plan

import (
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"strings"
)

type CleanupPlan interface {
	Plan(manifest manifest.Application, summary []cfclient.AppSummary) (pl Plan)
}

type cleanupPlan struct {
}

func (p cleanupPlan) Plan(manifest manifest.Application, summary []cfclient.AppSummary) (pl Plan) {
	for _, app := range summary {
		if strings.HasPrefix(app.Name, createDeleteName(manifest.Name, 0)) {
			pl = append(pl, NewCfCommand("delete", app.Name))
		}
	}
	return
}

func NewCleanupPlan() CleanupPlan {
	return cleanupPlan{
	}
}
