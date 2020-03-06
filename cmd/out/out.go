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

	"github.com/spf13/afero"
	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"github.com/springernature/halfpipe-deploy-resource/plan"
)

func main() {
	concourseRoot := os.Args[1]

	started := time.Now()

	logger := logger.NewLogger(os.Stderr)
	logger.Println("WARNING! YOU ARE RUNNING WITH CF CLI 7 WHICH IS IN BETA!")

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

	var p plan.Plan
	switch request.Params.Command {
	case "":
		panic("params.command must not be empty")
	case config.PUSH, config.CHECK, config.PROMOTE, config.DELETE, config.CLEANUP, config.DEPLOY_ROLLING, config.DELETE_TEST:
		fs := afero.Afero{Fs: afero.NewOsFs()}
		if err = plan.VerifyRequest(request); err != nil {
			break
		}

		p, err = plan.NewPlanner(
			manifest.NewManifestReadWrite(fs),
			fs,
		).Plan(request, concourseRoot)
	default:
		panic(fmt.Sprintf("Command '%s' not supported", request.Params.Command))
	}

	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	if err = p.Execute(plan.NewCFCliExecutor(&logger), &logger); err != nil {
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
