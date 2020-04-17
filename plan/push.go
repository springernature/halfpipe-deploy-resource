package plan

import (
	"fmt"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"strings"
)

type PushPlan struct {
	manifest manifest.Application
	request  Request
}

func (p PushPlan) Plan() (pl Plan, err error) {
	pl = append(pl, NewCfCommand("push").
		AddToArgs(p.getCandidateAppName()).
		AddToArgs("-f", p.request.Params.ManifestPath).
		AddToArgs("-p", p.request.Params.AppPath).
		AddToArgs("--no-route").
		AddToArgs("--no-start"))

	pl = append(pl, NewCfCommand("map-route").
		AddToArgs(p.getCandidateAppName()).
		AddToArgs(p.request.Params.TestDomain).
		AddToArgs("-n", p.getCandidateHostname()))

	pl = append(pl, NewCfCommand("start").
		AddToArgs(p.getCandidateAppName()))

	return
}

func (p PushPlan) getCandidateAppName() string {
	return fmt.Sprintf("%s-CANDIDATE", p.manifest.Name)
}

func (p PushPlan) getCandidateHostname() string {
	return strings.Join([]string{p.manifest.Name, p.request.Source.Space, "CANDIDATE"}, "-")
}

func NewPushPlan(manifest manifest.Application, request Request) PushPlan {
	return PushPlan{
		manifest: manifest,
		request:  request,
	}
}
