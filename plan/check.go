package plan

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"context"
	"fmt"
	cfclient "github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"time"
)

type CheckPlan interface {
	Plan(manifest manifestparser.Application, org, space string) (pl Plan)
}

type checkPlan struct {
}

func (p checkPlan) Plan(manifest manifestparser.Application, org, space string) (pl Plan) {
	desc := "Checking that all app instances are running"
	pl = append(pl, NewClientCommand(p.createFunc(createCandidateAppName(manifest.Name), org, space), desc))
	return
}

func (p checkPlan) createFunc(candidateAppName, org, space string) func(*cfclient.Client, *logger.CapturingWriter) error {
	return func(cfClient *cfclient.Client, logger *logger.CapturingWriter) error {
		ctx := context.Background()
		apps, err := getAppsInOrgSpace(ctx, cfClient, org, space)
		if err != nil {
			return err
		}

		appGuid := ""
		for _, app := range apps {
			if app.Name == candidateAppName {
				appGuid = app.GUID
				break
			}
		}
		if appGuid == "" {
			return fmt.Errorf("failed to find appGuid for app '%s'", candidateAppName)
		}

		for true {
			stats, err := cfClient.Processes.GetStatsForApp(ctx, appGuid, "web")
			if err != nil {
				return err
			}
			instances := stats.Stats

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
	return checkPlan{}
}
