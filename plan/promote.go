package plan

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"fmt"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"github.com/springernature/halfpipe-deploy-resource/config"
	"strings"
)

type PromotePlan interface {
	Plan(manifest manifestparser.Application, request config.Request, summary []*resource.App) (pl Plan)
}

type promotePlan struct {
	privateDomainsInOrg []*resource.Domain
}

func (p promotePlan) Plan(manifest manifestparser.Application, request config.Request, summary []*resource.App) (pl Plan) {
	currentLive, currentOld, currentDeletes := p.getPreviousAppState(manifest.Name, summary)

	pl = append(pl, p.addManifestRoutes(manifest)...)
	pl = append(pl, p.unmapTestRoute(manifest, request)...)
	pl = append(pl, p.renameOldApp(manifest, currentOld, currentDeletes)...)
	pl = append(pl, p.renameAndStopCurrentApp(manifest, currentLive)...)
	pl = append(pl, p.renameCandidateToLive(manifest))

	return
}

func (p promotePlan) renameOldApp(manifest manifestparser.Application, oldApp *resource.App, currentDeletes []*resource.App) (cmds []Command) {
	if oldApp != nil {
		nextI := 0
		for i := 1; i <= len(currentDeletes); i++ {
			found := false
			for _, currentDelete := range currentDeletes {
				if strings.HasSuffix(currentDelete.Name, fmt.Sprintf("-%d", i)) {
					found = true
				}
			}
			if !found {
				nextI = i
				break
			}
		}
		cmds = append(cmds, NewCfCommand("rename", createOldAppName(manifest.Name), createDeleteName(manifest.Name, nextI)))
	}

	return
}

func (p promotePlan) renameAndStopCurrentApp(manifest manifestparser.Application, currentLive *resource.App) (cmds []Command) {
	if currentLive != nil {
		cmds = append(cmds, NewCfCommand("rename", manifest.Name, createOldAppName(manifest.Name)))
		if currentLive.State == "STARTED" {
			cmds = append(cmds, NewCfCommand("stop", createOldAppName(manifest.Name)))
		}
	}
	return
}

func (p promotePlan) renameCandidateToLive(manifest manifestparser.Application) Command {
	return NewCfCommand("rename", createCandidateAppName(manifest.Name), manifest.Name)
}

func (p promotePlan) getPreviousAppState(manifestAppName string, summary []*resource.App) (currentLive, currentOld *resource.App, currentDeletes []*resource.App) {
	appFinder := func(name string, apps []*resource.App) (app *resource.App) {
		for _, app := range apps {
			if app.Name == name {
				return app
			}
		}
		return nil
	}

	deleteAppFinder := func(name string, apps []*resource.App) (deleteApps []*resource.App) {
		for _, app := range apps {
			if strings.HasPrefix(app.Name, name) {
				deleteApps = append(deleteApps, app)
			}
		}
		return
	}

	currentLive = appFinder(manifestAppName, summary)
	currentOld = appFinder(createOldAppName(manifestAppName), summary)
	currentDeletes = deleteAppFinder(createDeleteName(manifestAppName, 0), summary)
	return
}

func (p promotePlan) routes(man manifestparser.Application) (rs []string) {
	rawRoutes := []any{}

	if man.RemainingManifestFields["routes"] != nil {
		rawRoutes = man.RemainingManifestFields["routes"].([]any)
	}

	for _, r := range rawRoutes {
		route := r.(map[any]any)["route"].(string)
		rs = append(rs, route)
	}
	return rs
}

func (p promotePlan) addManifestRoutes(man manifestparser.Application) (cmds []Command) {
	isPrivateDomain := func(r string) bool {
		for _, domain := range p.privateDomainsInOrg {
			if r == domain.Name {
				return true
			}
		}
		return false
	}

	for _, route := range p.routes(man) {
		splitOnPath := strings.Split(route, "/")
		if isPrivateDomain(splitOnPath[0]) {
			var mapRoute Command
			if strings.Contains(route, "/") {
				mapRoute = NewCfCommand("map-route", createCandidateAppName(man.Name), splitOnPath[0], "--path", splitOnPath[1])
			} else {
				mapRoute = NewCfCommand("map-route", createCandidateAppName(man.Name), route)
			}

			cmds = append(cmds, mapRoute)
		} else {
			parts := strings.Split(route, ".")
			hostname := parts[0]
			domain := strings.Join(parts[1:], ".")
			if strings.Contains(domain, "/") {
				partsWithPath := strings.Split(domain, "/")
				domain = partsWithPath[0]
				path := strings.Join(partsWithPath[1:], "/")
				mapRoute := NewCfCommand("map-route", createCandidateAppName(man.Name), domain, "--hostname", hostname, "--path", path)
				cmds = append(cmds, mapRoute)
			} else {
				mapRoute := NewCfCommand("map-route", createCandidateAppName(man.Name), domain, "--hostname", hostname)
				cmds = append(cmds, mapRoute)
			}
		}
	}
	return
}

func (p promotePlan) unmapTestRoute(man manifestparser.Application, request config.Request) (cmds []Command) {
	if !man.NoRoute {
		unmapRoute := NewCfCommand("unmap-route", createCandidateAppName(man.Name), request.Params.TestDomain, "--hostname", createCandidateHostname(man, request))
		cmds = append(cmds, unmapRoute)
	}
	return
}

func NewPromotePlan(privateDomainsInOrg []*resource.Domain) PromotePlan {
	return promotePlan{
		privateDomainsInOrg: privateDomainsInOrg,
	}
}
