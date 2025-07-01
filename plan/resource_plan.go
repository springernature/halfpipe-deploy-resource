package plan

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"fmt"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"strings"
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
	logsPlan            LogsPlan
	appLintPlan         AppLintPlan
	ssoPlan             SSOPlan
}

func NewPlanner(manifestReaderWrite manifest.ReaderWriter, pushPlan PushPlan, checkPlan CheckPlan, promotePlan PromotePlan, cleanupPlan CleanupPlan, rollingDeployPlan RollingDeployPlan, deleteCandidatePlan DeleteCandidatePlan, logsPlan LogsPlan, appLintPlan AppLintPlan, ssoPlan SSOPlan) ResourcePlan {
	return planner{
		manifestReaderWrite: manifestReaderWrite,
		pushPlan:            pushPlan,
		promotePlan:         promotePlan,
		cleanupPlan:         cleanupPlan,
		checkPlan:           checkPlan,
		rollingDeployPlan:   rollingDeployPlan,
		deleteCandidatePlan: deleteCandidatePlan,
		logsPlan:            logsPlan,
		appLintPlan:         appLintPlan,
		ssoPlan:             ssoPlan,
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
		if err = p.updateManifestWithVarsAndLabels(request); err != nil {
			return
		}
		pl = append(pl, p.appLintPlan.Plan(appUnderDeployment, request.Source.Org, request.Source.Space)...)

		switch request.Params.Command {
		case config.PUSH:
			pl = append(pl, p.pushPlan.Plan(appUnderDeployment, request)...)
		case config.ROLLING_DEPLOY:
			pl = append(pl, p.rollingDeployPlan.Plan(appUnderDeployment, request)...)
		}
	case config.ALL:
		if err = p.updateManifestWithVarsAndLabels(request); err != nil {
			return
		}
		pl = append(pl, p.appLintPlan.Plan(appUnderDeployment, request.Source.Org, request.Source.Space)...)
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
	case config.LOGS:
		pl = append(pl, p.logsPlan.Plan(appUnderDeployment)...)
	case config.SSO:
		pl = append(pl, p.ssoPlan.Plan(request.Params.SSOHost)...)
	}

	return
}
func (p planner) readManifest(manifestPath string) (manifestparser.Manifest, error) {
	return p.manifestReaderWrite.ReadManifest(manifestPath)
}

func (p planner) updateManifestWithVarsAndLabels(request config.Request) (err error) {
	if len(request.Params.Vars) > 0 || request.Params.GitRefPath != "" {
		apps, e := p.readManifest(request.Params.ManifestPath)
		if e != nil {
			err = e
			return
		}

		env := make(map[any]any)
		metadata := make(map[any]any)
		labels := make(map[any]any)
		// We just assume the first app in the manifest is the app under deployment.
		// We lint that this is the case in the halfpipe linter.
		app := apps.Applications[0]
		if app.RemainingManifestFields == nil {
			app.RemainingManifestFields = map[string]any{}
		}
		if app.RemainingManifestFields["env"] != nil {
			env = app.RemainingManifestFields["env"].(map[any]any)
		}
		if app.RemainingManifestFields["metadata"] != nil {
			metadata = app.RemainingManifestFields["metadata"].(map[any]any)
			if metadata["labels"] != nil {
				labels = metadata["labels"].(map[any]any)
			}
		}

		if request.Metadata.GitRef != "" {
			env["GIT_REVISION"] = request.Metadata.GitRef
		}

		if request.Metadata.Version != "" {
			env["BUILD_VERSION"] = request.Metadata.Version
		}

		if request.Params.Team != "" || request.Params.GitUri != "" {
			if request.Params.Team != "" {
				labels["team"] = request.Params.Team
			}

			if request.Params.GitUri != "" {
				p1 := strings.Split(request.Params.GitUri, "/")
				if len(p1) == 2 {
					p2 := strings.Split(p1[1], ".")
					if len(p2) == 2 {
						labels["gitRepo"] = p2[0]
					}
				}
			}

			metadata["labels"] = labels
			app.RemainingManifestFields["metadata"] = metadata
		}

		for k, v := range request.Params.Vars {
			env[k] = v
		}

		p.otelEnv(env, app, request)

		app.RemainingManifestFields["env"] = env
		apps.Applications[0] = app

		if err = p.manifestReaderWrite.WriteManifest(request.Params.ManifestPath, apps); err != nil {
			return
		}
	}
	return
}

func (p planner) otelEnv(env map[any]any, app manifestparser.Application, request config.Request) {
	p.setIfNotOtelPresent(env, "OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	p.setIfNotOtelPresent(env, "OTEL_EXPORTER_OTLP_HEADERS", "X-Scope-OrgId=ee")
	p.setIfNotOtelPresent(env, "OTEL_SERVICE_NAME", app.Name)
	p.setIfNotOtelPresent(env, "OTEL_EXPORTER_OTLP_ENDPOINT", "http://opentelemetry-sink.tracing.springernature.io:9095")
	p.setIfNotOtelPresent(env, "OTEL_PROPAGATORS", "tracecontext")

	namespace := fmt.Sprintf("service.namespace=%s/%s", request.Source.Org, request.Source.Space)
	job := fmt.Sprintf("job=%s/%s/%s", request.Source.Org, request.Source.Space, app.Name)
	appName := fmt.Sprintf("cloudfoundry.app.name=%s", app.Name)
	org := fmt.Sprintf("cloudfoundry.app.org.name=%s", request.Source.Org)
	space := fmt.Sprintf("cloudfoundry.app.space.name=%s", request.Source.Space)
	p.setIfNotOtelPresent(env, "OTEL_RESOURCE_ATTRIBUTES", strings.Join([]string{namespace, job, appName, org, space}, ","))
}

func (p planner) setIfNotOtelPresent(env map[any]any, key string, defaultValue string) map[any]any {
	if _, found := env[key]; found {
		return env
	}
	env[key] = defaultValue
	return env
}
