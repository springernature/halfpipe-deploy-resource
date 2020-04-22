package plan

import (
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
)

type CleanupPlan interface {
	Plan(manifest manifest.Application, request Request, summary []cfclient.AppSummary) (pl Plan)
}

type cleanupPlan struct {
}

func (p cleanupPlan) Plan(manifest manifest.Application, request Request, summary []cfclient.AppSummary) (pl Plan) {
	return
}

func NewCleanupPlan() CleanupPlan {
	return cleanupPlan{
	}
}
