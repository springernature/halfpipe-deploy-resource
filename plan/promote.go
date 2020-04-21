package plan

import (
	"github.com/springernature/halfpipe-deploy-resource/manifest"
)

type PromotePlan interface {
	Plan(manifest manifest.Application, request Request) (pl Plan)
}

type promotePlan struct{}

func (p promotePlan) Plan(manifest manifest.Application, request Request) (pl Plan) {
	return nil
}

func (p promotePlan) promoteCommand(manifest manifest.Application, request Request) Command {
	return nil
}

func NewPromotePlan() PromotePlan {
	return promotePlan{}
}
