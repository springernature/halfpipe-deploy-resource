package plan

import (
	"fmt"
	"github.com/springernature/halfpipe-deploy-resource/config"
	"github.com/springernature/halfpipe-deploy-resource/manifest"
	"strings"
)

func createCandidateAppName(name string) string {
	return fmt.Sprintf("%s-CANDIDATE", name)
}

func createCandidateHostname(manifest manifest.Application, request config.Request) string {
	return strings.Join([]string{
		strings.Replace(manifest.Name, "_", "-", -1),
		strings.Replace(request.Source.Space, "_", "-", -1),
		"CANDIDATE"}, "-")
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
