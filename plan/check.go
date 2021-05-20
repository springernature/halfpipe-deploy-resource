package plan

import (
	"fmt"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"time"
)

type CheckPlan interface {
	Plan(manifest manifest.Application, org, space string) (pl Plan)
}

type checkPlan struct {
}

func (p checkPlan) Plan(manifest manifest.Application, org, space string) (pl Plan) {
	pl = append(pl, NewClientCommand(p.createFunc(createCandidateAppName(manifest.Name), org, space)))
	return
}

func (p checkPlan) getAppsInOrgSpace(client *cfclient.Client, orgName, spaceName string) (summary []cfclient.AppSummary, err error) {
	org, err := client.GetOrgByName(orgName)
	if err != nil {
		return
	}
	space, err := client.GetSpaceByName(spaceName, org.Guid)
	if err != nil {
		return
	}
	spaceSummary, err := space.Summary()
	if err != nil {
		return
	}
	summary = spaceSummary.Apps
	return
}

func (p checkPlan) createFunc(candidateAppName, org, space string) func(*cfclient.Client, *logger.CapturingWriter) error {
	return func(cfClient *cfclient.Client, logger *logger.CapturingWriter) error {
		apps, err := p.getAppsInOrgSpace(cfClient, org, space)
		if err != nil {
			return err
		}

		appGuid := ""
		for _, app := range apps {
			if app.Name == candidateAppName {
				appGuid = app.Guid
				break
			}
		}
		if appGuid == "" {
			return fmt.Errorf("failed to find appGuid for app '%s'", candidateAppName)
		}

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
