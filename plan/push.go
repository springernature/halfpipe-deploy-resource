package plan

import (
	"fmt"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"strings"
)

type PushPlan interface {
	Plan(manifest manifest.Application, request Request, dockerTag string) (pl Plan)
}

type pushPlan struct{}

func (p pushPlan) Plan(manifest manifest.Application, request Request, dockerTag string) (pl Plan) {
	pl = append(pl, p.pushCommand(manifest, request, dockerTag))

	if !manifest.NoRoute {
		pl = append(pl, NewCfCommand("map-route").
			AddToArgs(createCandidateAppName(manifest)).
			AddToArgs(request.Params.TestDomain).
			AddToArgs("-n", createCandidateHostname(manifest, request)))
	}

	if preStartArs := strings.Split(request.Params.PreStartCommand, "; "); request.Params.PreStartCommand != "" && len(preStartArs) > 0 {
		for _, prestartArg := range preStartArs {
			args := strings.Split(prestartArg, " ")[1:]
			pl = append(pl, NewCfCommand(args...))
		}
	}

	pl = append(pl, NewCompoundCommand(
		NewCfCommand("start").
			AddToArgs(createCandidateAppName(manifest)),
		NewCfCommand("logs",
			createCandidateAppName(manifest),
			"--recent",
		),
		func(log []byte) bool {
			return strings.Contains(string(log), `TIP: use 'cf logs`)
		}))

	return
}

func (p pushPlan) pushCommand(manifest manifest.Application, request Request, dockerTag string) Command {
	pushCommand := NewCfCommand("push").
		AddToArgs(createCandidateAppName(manifest)).
		AddToArgs("-f", request.Params.ManifestPath)

	if manifest.Docker.Image != "" {
		image := manifest.Docker.Image
		if dockerTag != "" {
			if strings.Contains(image, ":") {
				image = strings.Split(image, ":")[0]
			}
			image = fmt.Sprintf("%s:%s", image, dockerTag)
		}
		pushCommand = pushCommand.
			AddToArgs("--docker-image", image).
			AddToArgs("--docker-username", request.Params.DockerUsername).
			AddToEnv(fmt.Sprintf("CF_DOCKER_PASSWORD=%s", request.Params.DockerPassword))
	} else {
		pushCommand = pushCommand.AddToArgs("-p", request.Params.AppPath).
			AddToArgs("--no-route").
			AddToArgs("--no-start")
	}

	return pushCommand
}

func NewPushPlan() PushPlan {
	return pushPlan{}
}
