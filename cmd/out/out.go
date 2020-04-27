package main

import (
	"encoding/json"
	"fmt"
	"github.com/springernature/halfpipe-deploy-resource/fixes"
	"github.com/springernature/halfpipe-deploy-resource/logger"
	"io/ioutil"
	"os"
	"syscall"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/spf13/afero"
	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"github.com/springernature/halfpipe-deploy-resource/plan"
)

func main() {
	concourseRoot := os.Args[1]

	started := time.Now()

	logger := logger.NewLogger(os.Stderr)

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		logger.Println(err)
		syscall.Exit(1)
	}

	if err := ioutil.WriteFile("/tmp/request", data, 0777); err != nil {
		logger.Println(err)
		syscall.Exit(1)
	}

	request := plan.Request{}
	err = json.Unmarshal(data, &request)
	if err != nil {
		logger.Println(err)
		syscall.Exit(1)
	}

	cfClient, appsSummary, privateDomains, err := getApps(request)
	if err != nil {
		logger.Println(err)
		syscall.Exit(1)
	}

	var p plan.Plan
	switch request.Params.Command {
	case "":
		panic("params.command must not be empty")
	case config.PUSH, config.CHECK, config.PROMOTE, config.DELETE, config.CLEANUP:
		fs := afero.Afero{Fs: afero.NewOsFs()}
		if err = plan.VerifyRequest(request); err != nil {
			break
		}

		p, err = plan.NewPlanner(
			manifest.NewManifestReadWrite(fs),
			fs,
			plan.NewPushPlan(),
			plan.NewCheckPlan(),
			plan.NewPromotePlan(privateDomains),
			plan.NewCleanupPlan(),
		).Plan(request, concourseRoot, appsSummary)
	default:
		panic(fmt.Sprintf("Command '%s' not supported", request.Params.Command))
	}

	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	logger.Println(p.String())

	timeout, err := getTimeout(request)
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	if err = p.Execute(plan.NewCFCliExecutor(&logger), cfClient, &logger, timeout); err != nil {
		logger.Println(err)
		logger.Println("")
		for _, fix := range fixes.SuggestFix(logger.BytesWritten, request) {
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
			{Name: "Api", Value: request.Source.API},
			{Name: "Org", Value: request.Source.Org},
			{Name: "Space", Value: request.Source.Space},
			{Name: "Duration", Value: finished.Sub(started).String()},
		},
	}
	if err = json.NewEncoder(os.Stdout).Encode(response); err != nil {
		panic(err)
	}
}

func getTimeout(request plan.Request) (time.Duration, error) {
	if request.Params.Timeout == "" {
		return 5*time.Minute, nil
	}
	return time.ParseDuration(request.Params.Timeout)
}

func getApps(request plan.Request) (client *cfclient.Client, appSummary []cfclient.AppSummary, privateDomains []cfclient.Domain, err error) {
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
