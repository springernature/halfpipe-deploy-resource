package plan

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
)

type LogsPlan interface {
	Plan(manifest manifestparser.Application) (pl Plan)
}

type logsPlan struct {
}

func (p logsPlan) Plan(manifest manifestparser.Application) (pl Plan) {
	return Plan{
		NewCfCommand("logs", createCandidateAppName(manifest.Name), "--recent"),
	}
}

func NewLogsPlan() LogsPlan {
	return logsPlan{}
}
