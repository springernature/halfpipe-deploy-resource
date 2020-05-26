package plan

import (
	"fmt"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"time"
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

func (p checkPlan) createFunc(appGuid string) func(*cfclient.Client, *logger.CapturingWriter) error {
	return func(cfClient *cfclient.Client, logger *logger.CapturingWriter) error {
		for true {
			instances, err := cfClient.GetAppInstances(appGuid)
			if err != nil {
				return err
			}

			numRunning := 0
			for _, instance := range instances {
				if instance.State == "RUNNING" {
					numRunning += 1
				}
			}

			logger.Println(fmt.Sprintf(`%d/%d instances running`, numRunning, len(instances)))

			if len(instances) != numRunning {
				time.Sleep(10 * time.Second)
				continue
			}
			break
		}
		return nil
	}
}

func NewCheckPlan() CheckPlan {
	return checkPlan{
	}
}
