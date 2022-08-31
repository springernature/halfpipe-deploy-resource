package halfpipe_deploy_resource

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"fmt"
	"gopkg.in/yaml.v2"
)

func ParseManifest(manifest string) manifestparser.Manifest {
	man := manifestparser.Manifest{}
	err := yaml.Unmarshal([]byte(manifest), &man)
	if err != nil {
		panic(fmt.Sprintf("%s\nFailed to parse \n %s", err, manifest))
	}
	return man
}
