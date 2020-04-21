package plan

import (
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"strings"
)

type PromotePlan interface {
	Plan(manifest manifest.Application, request Request, summary []cfclient.AppSummary) (pl Plan)
}

type promotePlan struct{}

func (p promotePlan) Plan(manifest manifest.Application, request Request, summary []cfclient.AppSummary) (pl Plan) {
	currentLive, currentOld, currentDelete := p.getPreviousAppState(manifest.Name, summary)

	if currentOld.Name != "" {
		i := len(currentDelete)
		pl = append(pl, NewCfCommand("rename", createOldAppName(manifest.Name), createDeleteName(manifest.Name, i)))
	}

	if currentLive.Name != "" {
		pl = append(pl, NewCfCommand("rename", manifest.Name, createOldAppName(manifest.Name)))
		if currentLive.State == "started" {
			pl = append(pl, NewCfCommand("stop", createOldAppName(manifest.Name)))
		}
	}

	pl = append(pl, NewCfCommand("rename", createCandidateAppName(manifest.Name), manifest.Name))
	return pl
}

func (p promotePlan) getPreviousAppState(manifestAppName string, summary []cfclient.AppSummary) (currentLive, currentOld cfclient.AppSummary, currentDeletes []cfclient.AppSummary) {
	appFinder := func(name string, apps []cfclient.AppSummary) (app cfclient.AppSummary) {
		for _, app := range apps {
			if app.Name == name {
				return app
			}
		}
		return
	}

	deleteAppFinder := func(name string, apps []cfclient.AppSummary) (deleteApps []cfclient.AppSummary) {
		for _, app := range apps {
			if strings.HasPrefix(app.Name, name) {
				deleteApps = append(deleteApps, app)
			}
		}
		return
	}

	currentLive = appFinder(manifestAppName, summary)
	currentOld = appFinder(createOldAppName(manifestAppName), summary)
	currentDeletes = deleteAppFinder(createDeleteName(manifestAppName, 0), summary)
	return
}

func NewPromotePlan() PromotePlan {
	return promotePlan{}
}
