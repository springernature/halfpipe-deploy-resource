package plan

import (
	"fmt"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"strings"
)

type DynamicCleanupPlan interface {
	Plan(manifest manifest.Application, org, space string) (pl Plan)
}

type dynamicCleanupPlan struct {
}

func (p dynamicCleanupPlan) Plan(manifest manifest.Application, org, space string) (pl Plan) {
	desc := "Finding old apps to delete"
	pl = append(pl, NewClientCommand(p.createFunc(manifest.Name, org, space), desc))
	return
}

func (p dynamicCleanupPlan) getAppsInOrgSpace(client *cfclient.Client, orgName, spaceName string) (summary []cfclient.AppSummary, err error) {
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

func (p dynamicCleanupPlan) createFunc(appName, org, space string) func(*cfclient.Client, *logger.CapturingWriter) error {
	return func(cfClient *cfclient.Client, logger *logger.CapturingWriter) error {
		apps, err := p.getAppsInOrgSpace(cfClient, org, space)
		if err != nil {
			return err
		}

		for _, app := range apps {
			if strings.HasPrefix(app.Name, fmt.Sprintf("%s-DELETE", appName)) {
				logger.Println("Deleting", app.Name)
				err := cfClient.DeleteV3App(app.Guid)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func NewDynamicCleanupPlan() DynamicCleanupPlan {
	return dynamicCleanupPlan{
	}
}
