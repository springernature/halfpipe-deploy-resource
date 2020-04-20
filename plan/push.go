package plan

import (
	"fmt"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"strings"
)

type PushPlan struct {
	manifest  manifest.Application
	request   Request
	dockerTag string
}

func (p PushPlan) Plan() (pl Plan, err error) {
	pl = append(pl, p.pushCommand())

	pl = append(pl, NewCfCommand("map-route").
		AddToArgs(p.getCandidateAppName()).
		AddToArgs(p.request.Params.TestDomain).
		AddToArgs("-n", p.getCandidateHostname()))

	if preStartArs := strings.Split(p.request.Params.PreStartCommand, "; "); p.request.Params.PreStartCommand != "" && len(preStartArs) > 0 {
		for _, prestartArg := range preStartArs {
			args := strings.Split(prestartArg, " ")[1:]
			pl = append(pl, NewCfCommand(args...))
		}
	}

	pl = append(pl, NewCompoundCommand(
		NewCfCommand("start").
			AddToArgs(p.getCandidateAppName()),
		NewCfCommand("logs",
			p.getCandidateAppName(),
			"--recent",
		),
		func(log []byte) bool {
			return strings.Contains(string(log), `TIP: use 'cf logs`)
		}))

	return
}

func (p PushPlan) pushCommand() Command {
	pushCommand := NewCfCommand("push").
		AddToArgs(p.getCandidateAppName()).
		AddToArgs("-f", p.request.Params.ManifestPath)

	if p.manifest.Docker.Image != "" {
		image := p.manifest.Docker.Image
		if p.dockerTag != "" {
			if strings.Contains(image, ":") {
				image = strings.Split(image, ":")[0]
			}
			image = fmt.Sprintf("%s:%s", image, p.dockerTag)
		}
		pushCommand = pushCommand.
			AddToArgs("--docker-image", image).
			AddToArgs("--docker-username", p.request.Params.DockerUsername).
			AddToEnv(fmt.Sprintf("CF_DOCKER_PASSWORD=%s", p.request.Params.DockerPassword))
	} else {
		pushCommand = pushCommand.AddToArgs("-p", p.request.Params.AppPath).
			AddToArgs("--no-route").
			AddToArgs("--no-start")
	}

	return pushCommand
}

func (p PushPlan) getCandidateAppName() string {
	return fmt.Sprintf("%s-CANDIDATE", p.manifest.Name)
}

func (p PushPlan) getCandidateHostname() string {
	return strings.Join([]string{p.manifest.Name, p.request.Source.Space, "CANDIDATE"}, "-")
}

func NewPushPlan(manifest manifest.Application, request Request, dockerTag string) PushPlan {
	return PushPlan{
		manifest:  manifest,
		request:   request,
		dockerTag: dockerTag,
	}
}
