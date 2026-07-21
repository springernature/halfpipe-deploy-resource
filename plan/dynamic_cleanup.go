package plan

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"context"
	"fmt"
	cfclient "github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"strings"
)

type DynamicCleanupPlan interface {
	Plan(manifest manifestparser.Application, org, space string) (pl Plan)
}

type dynamicCleanupPlan struct {
}

func (p dynamicCleanupPlan) Plan(manifest manifestparser.Application, org, space string) (pl Plan) {
	desc := "Finding old apps to delete"
	pl = append(pl, NewClientCommand(p.createFunc(manifest.Name, org, space), desc))
	return
}

func (p dynamicCleanupPlan) createFunc(appName, org, space string) func(*cfclient.Client, *logger.CapturingWriter) error {
	return func(cfClient *cfclient.Client, logger *logger.CapturingWriter) error {
		ctx := context.Background()
		apps, err := getAppsInOrgSpace(ctx, cfClient, org, space)
		if err != nil {
			return err
		}

		for _, app := range apps {
			if strings.HasPrefix(app.Name, fmt.Sprintf("%s-DELETE", appName)) {
				logger.Println("Deleting", app.Name)
				_, err := cfClient.Applications.Delete(ctx, app.GUID)
				if err != nil {
					return err
				}
			}
		}
		logger.Println("OK")
		return nil
	}
}

func NewDynamicCleanupPlan() DynamicCleanupPlan {
	return dynamicCleanupPlan{}
}
