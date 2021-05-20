package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/gookit/color"
	"github.com/springernature/halfpipe-deploy-resource/fixes"
	"github.com/springernature/halfpipe-deploy-resource/logger"

	"github.com/cloudfoundry-community/go-cfclient"
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

	requestConfig, err := config.NewRequestReader(os.Args, environmentToMap(), os.Stdin, afero.Afero{Fs: afero.NewOsFs()}).ReadRequest()
	if err != nil {
		logger.Println(err)
		syscall.Exit(1)
	}

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
	case config.PUSH, config.CHECK, config.PROMOTE, config.DELETE, config.CLEANUP, config.ROLLING_DEPLOY, config.DELETE_CANDIDATE, config.ALL:
		fs := afero.Afero{Fs: afero.NewOsFs()}
		if requestConfig.Params.CliVersion == "" {
			requestConfig.Params.CliVersion = "cf6"
		}

		p, err = plan.NewPlanner(manifest.NewManifestReadWrite(fs), plan.NewPushPlan(), plan.NewCheckPlan(), plan.NewPromotePlan(privateDomains), plan.NewCleanupPlan(), plan.NewRollingDeployPlan(), plan.NewDeleteCandidatePlan()).Plan(requestConfig, appsSummary)
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

	if err = p.Execute(plan.NewCFCliExecutor(&logger, requestConfig), cfClient, &logger, timeout); err != nil {
		logger.Println(err)
		logger.Println("")
		for _, fix := range fixes.SuggestFix(logger.BytesWritten, requestConfig) {
			logger.Println(fix)
		}

		os.Exit(1)
	}

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

func getApps(request config.Request) (client *cfclient.Client, appSummary []cfclient.AppSummary, privateDomains []cfclient.Domain, err error) {
	c := &cfclient.Config{
		ApiAddress: request.Source.API,
		Username:   request.Source.Username,
		Password:   request.Source.Password,
	}
	client, err = cfclient.NewClient(c)
	if err != nil {
		return
	}
	org, err := client.GetOrgByName(request.Source.Org)
	if err != nil {
		return
	}
	space, err := client.GetSpaceByName(request.Source.Space, org.Guid)
	if err != nil {
		return
	}
	spaceSummary, err := space.Summary()
	if err != nil {
		return
	}
	appSummary = spaceSummary.Apps

	privateDomains, err = org.ListPrivateDomains()
	if err != nil {
		return
	}

	return
}
