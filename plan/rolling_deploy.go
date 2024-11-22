package plan

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"fmt"
	"github.com/springernature/halfpipe-deploy-resource/config"
	"strings"
)

type RollingDeployPlan interface {
	Plan(manifest manifestparser.Application, request config.Request) (pl Plan)
}

type rollingDeployPlan struct{}

func (p rollingDeployPlan) Plan(manifest manifestparser.Application, request config.Request) (pl Plan) {
	pushCommand := NewCfCommand("push").
		AddToArgs("--manifest", request.Params.ManifestPath).
		AddToArgs("--strategy", "rolling")

	if manifest.Docker != nil && manifest.Docker.Image != "" {
		image := manifest.Docker.Image
		if request.Metadata.DockerTag != "" {
			if strings.Contains(image, ":") {
				image = strings.Split(image, ":")[0]
			}
			image = fmt.Sprintf("%s:%s", image, request.Metadata.DockerTag)
		}
		pushCommand = pushCommand.
			AddToArgs("--docker-image", image).
			AddToArgs("--docker-username", request.Params.DockerUsername).
			AddToEnv(fmt.Sprintf("CF_DOCKER_PASSWORD=%s", request.Params.DockerPassword))
	} else {
		pushCommand = pushCommand.AddToArgs("--path", request.Params.AppPath)
	}

	pl = append(pl, NewCompoundCommand(pushCommand, NewCfCommand("logs",
		manifest.Name,
		"--recent",
	), func(log []byte) bool {
		return strings.Contains(string(log), `--recent' for more information`)
	}, true))

	return
}

func NewRollingDeployPlan() RollingDeployPlan {
	return rollingDeployPlan{}
}
