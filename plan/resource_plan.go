package plan

import (
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
)

type ResourcePlan interface {
	Plan(request config.Request, appsSummary []cfclient.AppSummary) (plan Plan, err error)
}

type planner struct {
	manifestReaderWrite manifest.ReaderWriter
	pushPlan            PushPlan
	rollingDeployPlan   RollingDeployPlan
	promotePlan         PromotePlan
	cleanupPlan         CleanupPlan
	checkPlan           CheckPlan
	deleteCandidatePlan DeleteCandidatePlan
}

func NewPlanner(manifestReaderWrite manifest.ReaderWriter, pushPlan PushPlan, checkPlan CheckPlan, promotePlan PromotePlan, cleanupPlan CleanupPlan, rollingDeployPlan RollingDeployPlan, deleteCandidatePlan DeleteCandidatePlan) ResourcePlan {
	return planner{
		manifestReaderWrite: manifestReaderWrite,
		pushPlan:            pushPlan,
		promotePlan:         promotePlan,
		cleanupPlan:         cleanupPlan,
		checkPlan:           checkPlan,
		rollingDeployPlan:   rollingDeployPlan,
		deleteCandidatePlan: deleteCandidatePlan,
	}
}

func (p planner) Plan(request config.Request, appsSummary []cfclient.AppSummary) (pl Plan, err error) {
	// Here we assume that the request is complete.
	// It has already been verified.

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

		switch request.Params.Command {
		case config.PUSH:
			pl = append(pl, p.pushPlan.Plan(appUnderDeployment, request)...)
		case config.ROLLING_DEPLOY:
			pl = append(pl, p.rollingDeployPlan.Plan(appUnderDeployment, request)...)
		}
	case config.CHECK:
		// We dont actually need to login for this as we are using a cf client for this specific task..
		pl = p.checkPlan.Plan(appUnderDeployment, appsSummary)
	case config.PROMOTE:
		pl = append(pl, p.promotePlan.Plan(appUnderDeployment, request, appsSummary)...)
	case config.CLEANUP, config.DELETE:
		pl = append(pl, p.cleanupPlan.Plan(appUnderDeployment, appsSummary)...)
	case config.DELETE_CANDIDATE:
		pl = append(pl, p.deleteCandidatePlan.Plan(appUnderDeployment, appsSummary)...)
	case config.ALL:
		if err = p.updateManifestWithVars(request); err != nil {
			return
		}

		pl = append(pl, p.pushPlan.Plan(appUnderDeployment, request)...)
		pl = append(pl, NewShellCommand("sleep", "10"))
		pl = append(pl, p.checkPlan.Plan(appUnderDeployment, appsSummary)...)
		pl = append(pl, p.promotePlan.Plan(appUnderDeployment, request, appsSummary)...)
		pl = append(pl, p.cleanupPlan.Plan(appUnderDeployment, appsSummary)...)
	}

	return
}

func (p planner) readManifest(manifestPath string) (manifest.Manifest, error) {
	return p.manifestReaderWrite.ReadManifest(manifestPath)
}

func (p planner) updateManifestWithVars(request config.Request) (err error) {
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

		if request.Metadata.GitRef != "" {
			app.EnvironmentVariables["GIT_REVISION"] = request.Metadata.GitRef
		}

		if request.Metadata.Version != "" {
			app.EnvironmentVariables["BUILD_VERSION"] = request.Metadata.Version
		}

		if err = p.manifestReaderWrite.WriteManifest(request.Params.ManifestPath, app); err != nil {
			return
		}
	}
	return
}
