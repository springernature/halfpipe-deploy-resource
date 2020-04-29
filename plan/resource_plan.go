package plan

import (
	"github.com/cloudfoundry-community/go-cfclient"
	"path"

	"github.com/spf13/afero"
	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"strings"
)

type ResourcePlan interface {
	Plan(request Request, concourseRoot string, appsSummary []cfclient.AppSummary) (plan Plan, err error)
}

type planner struct {
	manifestReaderWrite manifest.ReaderWriter
	fs                  afero.Afero
	pushPlan            PushPlan
	rollingDeployPlan   RollingDeployPlan
	promotePlan         PromotePlan
	cleanupPlan         CleanupPlan
	checkPlan           CheckPlan
}

func NewPlanner(manifestReaderWrite manifest.ReaderWriter, fs afero.Afero, pushPlan PushPlan, checkPlan CheckPlan, promotePlan PromotePlan, cleanupPlan CleanupPlan, rollingDeployPlan RollingDeployPlan) ResourcePlan {
	return planner{
		manifestReaderWrite: manifestReaderWrite,
		fs:                  fs,
		pushPlan:            pushPlan,
		promotePlan:         promotePlan,
		cleanupPlan:         cleanupPlan,
		checkPlan:           checkPlan,
		rollingDeployPlan:   rollingDeployPlan,
	}
}

func (p planner) setFullPathInRequest(request Request, concourseRoot string) Request {
	updatedRequest := request

	updatedRequest.Params.ManifestPath = path.Join(concourseRoot, updatedRequest.Params.ManifestPath)

	if updatedRequest.Params.AppPath != "" {
		updatedRequest.Params.AppPath = path.Join(concourseRoot, updatedRequest.Params.AppPath)
	}

	if updatedRequest.Params.DockerTag != "" {
		updatedRequest.Params.DockerTag = path.Join(concourseRoot, updatedRequest.Params.DockerTag)
	}

	if request.Params.GitRefPath != "" {
		updatedRequest.Params.GitRefPath = path.Join(concourseRoot, request.Params.GitRefPath)
	}

	if request.Params.BuildVersionPath != "" {
		updatedRequest.Params.BuildVersionPath = path.Join(concourseRoot, request.Params.BuildVersionPath)
	}

	return updatedRequest
}

func (p planner) Plan(request Request, concourseRoot string, appsSummary []cfclient.AppSummary) (pl Plan, err error) {
	// Here we assume that the request is complete.
	// It has already been verified in out.go with the help of requests.VerifyRequest.

	// Here we update the paths to take into account concourse root
	request = p.setFullPathInRequest(request, concourseRoot)

	readManifest, err := p.readManifest(request.Params.ManifestPath)
	if err != nil {
		// todo: test this
		return
	}

	// We lint that there is only one app.
	appUnderDeployment := readManifest.Applications[0]

	pl = append(pl, NewCfCommand("--version"))

	pl = append(pl, NewCfCommand("login",
		"-a", request.Source.API,
		"-u", request.Source.Username,
		"-p", request.Source.Password,
		"-o", request.Source.Org,
		"-s", request.Source.Space))

	switch request.Params.Command {
	case config.PUSH, config.ROLLING_DEPLOY:
		if err = p.updateManifestWithVars(request); err != nil {
			return
		}
		var dockerTag string
		if request.Params.DockerTag != "" {
			content, e := p.fs.ReadFile(request.Params.DockerTag)
			if e != nil {
				err = e
				return
			}
			dockerTag = string(content)
		}

		switch request.Params.Command {
		case config.PUSH:
			pl = append(pl, p.pushPlan.Plan(appUnderDeployment, request, dockerTag)...)
		case config.ROLLING_DEPLOY:
			pl = append(pl, p.rollingDeployPlan.Plan(appUnderDeployment, request, dockerTag)...)
		}
	case config.CHECK:
		// We dont actually need to login for this as we are using a cf client for this specific task..
		pl = p.checkPlan.Plan(appUnderDeployment, appsSummary)
	case config.PROMOTE:
		pl = append(pl, p.promotePlan.Plan(appUnderDeployment, request, appsSummary)...)
	case config.CLEANUP, config.DELETE:
		pl = append(pl, p.cleanupPlan.Plan(appUnderDeployment, appsSummary)...)
	}

	return
}

func (p planner) readManifest(manifestPath string) (manifest.Manifest, error) {
	return p.manifestReaderWrite.ReadManifest(manifestPath)
}

func (p planner) updateManifestWithVars(request Request) (err error) {
	if len(request.Params.Vars) > 0 || request.Params.GitRefPath != "" {
		apps, e := p.readManifest(request.Params.ManifestPath)
		if e != nil {
			err = e
			return
		}

		// We just assume the first app in the manifest is the app under deployment.
		// We lint that this is the case in the halfpipe linter.
		app := apps.Applications[0]
		if len(app.EnvironmentVariables) == 0 {
			app.EnvironmentVariables = map[string]string{}
		}

		for key, value := range request.Params.Vars {
			app.EnvironmentVariables[key] = value
		}

		if request.Params.GitRefPath != "" {
			ref, errRead := p.readFile(request.Params.GitRefPath)
			if errRead != nil {
				err = errRead
				return
			}
			app.EnvironmentVariables["GIT_REVISION"] = ref
		}

		if request.Params.BuildVersionPath != "" {
			version, errRead := p.readFile(request.Params.BuildVersionPath)
			if errRead != nil {
				err = errRead
				return
			}
			app.EnvironmentVariables["BUILD_VERSION"] = version
		}

		if err = p.manifestReaderWrite.WriteManifest(request.Params.ManifestPath, app); err != nil {
			return
		}
	}
	return
}

func (p planner) readFile(gitRefPath string) (ref string, err error) {
	bytes, err := p.fs.ReadFile(gitRefPath)
	if err != nil {
		return
	}
	ref = strings.TrimSpace(string(bytes))
	return
}
