package plan

import (
	"fmt"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"strings"
)

type RollingDeployPlan interface {
	Plan(manifest manifest.Application, request Request, dockerTag string) (pl Plan)
}

type rollingDeployPlan struct{}

func (p rollingDeployPlan) Plan(manifest manifest.Application, request Request, dockerTag string) (pl Plan) {
	pushCommand := NewCfCommand("push").
		AddToArgs("--manifest", request.Params.ManifestPath).
		AddToArgs("--strategy", "rolling").
		AddToArgs("--path", request.Params.AppPath)

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
	}

	pl = append(pl, NewCompoundCommand(pushCommand, NewCfCommand("logs",
		manifest.Name,
		"--recent",
	), func(log []byte) bool {
		return strings.Contains(string(log), `TIP: use 'cf logs`)
	}))

	return
}

func NewRollingDeployPlan() RollingDeployPlan {
	return rollingDeployPlan{}
}
