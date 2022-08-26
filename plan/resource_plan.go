package plan

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
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
	newManifest, err := p.readManifest(request.Params.ManifestPath)
	if err != nil {
		return
	}

	// We lint that there is only one app.
	appUnderDeployment := newManifest.Applications[0]

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
	case config.ALL:
		if err = p.updateManifestWithVars(request); err != nil {
			return
		}
		pl = append(pl, p.pushPlan.Plan(appUnderDeployment, request)...)
		pl = append(pl, p.checkPlan.Plan(appUnderDeployment, request.Source.Org, request.Source.Space)...)
		pl = append(pl, p.promotePlan.Plan(appUnderDeployment, request, appsSummary)...)
		pl = append(pl, NewDynamicCleanupPlan().Plan(appUnderDeployment, request.Source.Org, request.Source.Space)...)
	case config.CHECK:
		// We dont actually need to login for this as we are using a cf client for this specific task..
		pl = p.checkPlan.Plan(appUnderDeployment, request.Source.Org, request.Source.Space)
	case config.PROMOTE:
		pl = append(pl, p.promotePlan.Plan(appUnderDeployment, request, appsSummary)...)
	case config.CLEANUP, config.DELETE:
		pl = append(pl, p.cleanupPlan.Plan(appUnderDeployment, appsSummary)...)
	case config.DELETE_CANDIDATE:
		pl = append(pl, p.deleteCandidatePlan.Plan(appUnderDeployment, appsSummary)...)
	}

	return
}
func (p planner) readManifest(manifestPath string) (manifestparser.Manifest, error) {
	return p.manifestReaderWrite.ReadManifest(manifestPath)
}

func (p planner) updateManifestWithVars(request config.Request) (err error) {
	if len(request.Params.Vars) > 0 || request.Params.GitRefPath != "" {
		apps, e := p.readManifest(request.Params.ManifestPath)
		if e != nil {
			err = e
			return
		}

		env := make(map[string]any)

		// We just assume the first app in the manifest is the app under deployment.
		// We lint that this is the case in the halfpipe linter.
		app := apps.Applications[0]
		if app.RemainingManifestFields == nil {
			app.RemainingManifestFields = map[string]any{}
		}
		if app.RemainingManifestFields["env"] != nil {
			env = app.RemainingManifestFields["env"].(map[string]any)
		}

		if request.Metadata.GitRef != "" {
			env["GIT_REVISION"] = request.Metadata.GitRef
		}

		if request.Metadata.Version != "" {
			env["BUILD_VERSION"] = request.Metadata.Version
		}

		for k, v := range request.Params.Vars {
			env[k] = v
		}
		app.RemainingManifestFields["env"] = env
		apps.Applications[0] = app

		if err = p.manifestReaderWrite.WriteManifest(request.Params.ManifestPath, apps); err != nil {
			return
		}
	}
	return
}
