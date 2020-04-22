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
	currentLive, currentOld, currentDeletes := p.getPreviousAppState(manifest.Name, summary)

	pl = append(pl, p.renameOldApp(manifest, currentOld, currentDeletes)...)
	pl = append(pl, p.renameAndStopCurrentApp(manifest, currentLive)...)
	pl = append(pl, p.renameCandidateToLive(manifest))

	return
}

func (p promotePlan) renameOldApp(manifest manifest.Application, oldApp cfclient.AppSummary, currentDeletes []cfclient.AppSummary) (cmds []Command) {
	if oldApp.Name != "" {
		i := len(currentDeletes)
		cmds = append(cmds, NewCfCommand("rename", createOldAppName(manifest.Name), createDeleteName(manifest.Name, i)))
	}

	return
}

func (p promotePlan) renameAndStopCurrentApp(manifest manifest.Application, currentLive cfclient.AppSummary) (cmds []Command) {
	if currentLive.Name != "" {
		cmds = append(cmds, NewCfCommand("rename", manifest.Name, createOldAppName(manifest.Name)))
		if currentLive.State == "started" {
			cmds = append(cmds, NewCfCommand("stop", createOldAppName(manifest.Name)))
		}
	}
	return
}

func (p promotePlan) renameCandidateToLive(manifest manifest.Application) Command {
	return NewCfCommand("rename", createCandidateAppName(manifest.Name), manifest.Name)
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
