package plan

import (
	"fmt"
	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"strconv"
	"strings"
)

type PushPlan interface {
	Plan(manifest manifest.Application, request config.Request) (pl Plan)
}

type pushPlan struct{}

func (p pushPlan) Plan(manifest manifest.Application, request config.Request) (pl Plan) {
	pl = append(pl, p.pushCommand(manifest, request))

	if !manifest.NoRoute {
		pl = append(pl, NewCfCommand("map-route").
			AddToArgs(createCandidateAppName(manifest.Name)).
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
			AddToArgs(createCandidateAppName(manifest.Name)),
		NewCfCommand("logs",
			createCandidateAppName(manifest.Name),
			"--recent",
		),
		func(log []byte) bool {
			return strings.Contains(string(log), `--recent' for more information`) ||
				strings.Contains(string(log), `insufficient resources: memory`)
		}))

	return
}

func (p pushPlan) pushCommand(manifest manifest.Application, request config.Request) Command {
	pushCommand := NewCfCommand("push").
		AddToArgs(createCandidateAppName(manifest.Name)).
		AddToArgs("-f", request.Params.ManifestPath)

	if request.Params.Instances != 0 {
		pushCommand = pushCommand.AddToArgs("-i", strconv.Itoa(request.Params.Instances))
	}

	if manifest.Docker.Image == "" {
		pushCommand = pushCommand.AddToArgs("-p", request.Params.AppPath)
	} else {
		pushCommand = pushCommand.
			AddToArgs("--docker-image", p.formatDockerImage(manifest, request.Metadata.DockerTag)).
			AddToArgs("--docker-username", request.Params.DockerUsername).
			AddToEnv(fmt.Sprintf("CF_DOCKER_PASSWORD=%s", request.Params.DockerPassword))
	}

	pushCommand = pushCommand.
		AddToArgs("--no-route").
		AddToArgs("--no-start")
	return pushCommand
}

func (p pushPlan) formatDockerImage(man manifest.Application, dockerTag string) string {
	image := man.Docker.Image
	if dockerTag != "" {
		if strings.Contains(image, ":") {
			image = strings.Split(image, ":")[0]
		}
		image = fmt.Sprintf("%s:%s", image, dockerTag)
	}
	return image
}

func NewPushPlan() PushPlan {
	return pushPlan{}
}
