package plan

import (
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
)

type CheckPlan interface {
	Plan(manifest manifest.Application, summary []cfclient.AppSummary) (pl Plan)
}

type checkPlan struct {
}

func (p checkPlan) Plan(manifest manifest.Application, summary []cfclient.AppSummary) (pl Plan) {
	guid := ""
	for _, app := range summary {
		if app.Name == createCandidateAppName(manifest.Name) {
			guid = app.Guid
		}
	}

	pl = append(pl, NewClientCommand(p.createFunc(guid)))

	return
}

func (p checkPlan) createFunc(appGuid string) func(client cfclient.Client, logger *logger.CapturingWriter) error {
	return func(client cfclient.Client, logger *logger.CapturingWriter) error {
		logger.Println("Checking that all app instances are in running state")
		logger.Println(appGuid)
		logger.Println("Yay, this is actually called")
		app, _ := client.GetAppByGuid(appGuid)
		logger.Println(app.Name)
		logger.Println(app.Instances)
		logger.Println(app.State)
		return nil
	}
}

func NewCheckPlan() CheckPlan {
	return checkPlan{
	}
}
