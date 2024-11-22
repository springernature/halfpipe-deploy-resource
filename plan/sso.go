package plan

import (
	"fmt"
	"strings"
)

type SSOPlan interface {
	Plan(ssoHost string) (pl Plan)
}

type ssoPlan struct {
}

func (s ssoPlan) Plan(ssoHost string) (plan Plan) {
	plan = append(plan, NewCompoundCommand(
		NewCfCommand("service", "sso"),
		NewCfCommand("create-user-provided-service", "sso", "-r", "https://ee-sso.public.springernature.app"),
		func(log []byte) bool {
			return strings.Contains(string(log), "Service instance sso not found")
		},
		false,
	))

	//cf8 bind-route-service public.springernature.app -n $SSO_HOST sso;

	plan = append(plan, NewCompoundCommand(
		NewCfCommand("route", "public.springernature.app", "-n", ssoHost),
		NewCfCommand("create-route", "public.springernature.app", "-n", ssoHost),
		func(log []byte) bool {
			return strings.Contains(string(log), fmt.Sprintf("Route with host '%s'", ssoHost))
		},
		false,
	))

	plan = append(plan, NewCfCommand("bind-route-service", "public.springernature.app", "-n", ssoHost))

	return
}

func NewSSOPlan() SSOPlan {
	return ssoPlan{}
}
