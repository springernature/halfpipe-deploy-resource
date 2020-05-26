package plan

import (
	"fmt"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"strings"
)

func createCandidateAppName(name string) string {
	return fmt.Sprintf("%s-CANDIDATE", name)
}

func createCandidateHostname(manifest manifest.Application, request Request) string {
	return strings.Join([]string{manifest.Name, request.Source.Space, "CANDIDATE"}, "-")
}

func createOldAppName(name string) string {
	return fmt.Sprintf("%s-OLD", name)
}

func createDeleteName(name string, index int) string {
	if index == 0 {
		return fmt.Sprintf("%s-DELETE", name)
	}
	return fmt.Sprintf("%s-DELETE-%d", name, index)
}
