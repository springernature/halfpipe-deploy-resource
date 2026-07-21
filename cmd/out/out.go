package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/springernature/halfpipe-deploy-resource/cmd/out/check_resource"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/gookit/color"
	"github.com/springernature/halfpipe-deploy-resource/fixes"
	"github.com/springernature/halfpipe-deploy-resource/logger"

	cfclient "github.com/cloudfoundry/go-cfclient/v3/client"
	cfconfig "github.com/cloudfoundry/go-cfclient/v3/config"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"github.com/spf13/afero"
	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"github.com/springernature/halfpipe-deploy-resource/plan"
)

func environmentToMap() map[string]string {
	env := make(map[string]string)
	for _, element := range os.Environ() {
		parts := strings.SplitN(element, "=", 2)
		env[parts[0]] = parts[1]
	}
	return env
}

func main() {
	started := time.Now()
	logger := logger.NewLogger(os.Stderr)
	fs := afero.Afero{Fs: afero.NewOsFs()}
	requestConfig, err := config.NewRequestReader(
		os.Args,
		environmentToMap(),
		os.Stdin,
		fs,
		manifest.NewManifestReadWrite(fs)).ReadRequest()
	if err != nil {
		logger.Println(err)
		syscall.Exit(1)
	}

	if requestConfig.Params.Command == "check" {
		// Here be dragons.
		// This is just to make sure the pipeline where we build and test the resource is correct..
		check_resource.CheckResource(os.Args, logger)
		json.NewEncoder(os.Stdout).Encode(plan.Response{})
		syscall.Exit(0)
	}

	metrics := plan.NewMetrics(requestConfig, "https://aggregationgateway.k8s.springernature.io/")

	cfClient, appsSummary, privateDomains, err := getApps(requestConfig)
	if err != nil {
		errStr := fmt.Sprintf("Unable to login to api: %s, org: %s, space: %s with user %s", requestConfig.Source.API, requestConfig.Source.Org, requestConfig.Source.Space, requestConfig.Source.Username)
		logger.Println(errStr)
		logger.Println(err)
		syscall.Exit(1)
	}

	var p plan.Plan
	switch requestConfig.Params.Command {
	case "":
		panic("params.command must not be empty")
	case config.PUSH, config.CHECK, config.PROMOTE, config.DELETE, config.CLEANUP, config.ROLLING_DEPLOY, config.DELETE_CANDIDATE, config.STOP_CANDIDATE, config.ALL, config.LOGS, config.SSO:

		if requestConfig.Params.CliVersion == "" {
			requestConfig.Params.CliVersion = "cf6"
		}

		p, err = plan.NewPlanner(manifest.NewManifestReadWrite(fs), plan.NewPushPlan(), plan.NewCheckPlan(), plan.NewPromotePlan(privateDomains), plan.NewCleanupPlan(), plan.NewRollingDeployPlan(), plan.NewDeleteCandidatePlan(), plan.NewStopCandidatePlan(), plan.NewLogsPlan(), plan.NewCheckLabelsPlan(), plan.NewSSOPlan()).Plan(requestConfig, appsSummary)
	default:
		panic(fmt.Sprintf("Command '%s' not supported", requestConfig.Params.Command))
	}

	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	logger.Println(color.New(color.FgGreen).Sprintf("%s", p.String()))

	timeout, err := getTimeout(requestConfig)
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	if err = p.Execute(plan.NewCFCliExecutor(&logger, requestConfig), cfClient, &logger, timeout, requestConfig.Metadata.IsActions); err != nil {
		logger.Println(err)
		logger.Println("")
		for _, fix := range fixes.SuggestFix(logger.BytesWritten, requestConfig) {
			logger.Println(fix)
		}
		metrics.Failure()
		os.Exit(1)
	}

	metrics.Success()
	finished := time.Now()

	response := plan.Response{
		Version: plan.Version{
			Timestamp: finished,
		},
		Metadata: []plan.MetadataPair{
			{Name: "Api", Value: requestConfig.Source.API},
			{Name: "Org", Value: requestConfig.Source.Org},
			{Name: "Space", Value: requestConfig.Source.Space},
			{Name: "Duration", Value: finished.Sub(started).String()},
		},
	}
	if !requestConfig.Metadata.IsActions {
		if err = json.NewEncoder(os.Stdout).Encode(response); err != nil {
			panic(err)
		}
	}
}

func getTimeout(request config.Request) (time.Duration, error) {
	if request.Params.Timeout == "" {
		return 15 * time.Minute, nil
	}
	return time.ParseDuration(request.Params.Timeout)
}

func getApps(request config.Request) (client *cfclient.Client, appSummary []*resource.App, privateDomains []*resource.Domain, err error) {
	ctx := context.Background()

	c, err := cfconfig.New(request.Source.API, cfconfig.UserPassword(request.Source.Username, request.Source.Password))
	if err != nil {
		return
	}
	client, err = cfclient.New(c)
	if err != nil {
		return
	}

	orgOpts := cfclient.NewOrganizationListOptions()
	orgOpts.Names = cfclient.Filter{Values: []string{request.Source.Org}}
	org, err := client.Organizations.Single(ctx, orgOpts)
	if err != nil {
		return
	}

	spaceOpts := cfclient.NewSpaceListOptions()
	spaceOpts.Names = cfclient.Filter{Values: []string{request.Source.Space}}
	spaceOpts.OrganizationGUIDs = cfclient.Filter{Values: []string{org.GUID}}
	space, err := client.Spaces.Single(ctx, spaceOpts)
	if err != nil {
		return
	}

	appOpts := cfclient.NewAppListOptions()
	appOpts.SpaceGUIDs = cfclient.Filter{Values: []string{space.GUID}}
	appSummary, err = client.Applications.ListAll(ctx, appOpts)
	if err != nil {
		return
	}

	privateDomains, err = client.Domains.ListForOrganizationAll(ctx, org.GUID, cfclient.NewDomainListOptions())
	if err != nil {
		return
	}

	return
}
