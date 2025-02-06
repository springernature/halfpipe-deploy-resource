package plan

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"fmt"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource/logger"
)

type AppLintPlan interface {
	Plan(manifest manifestparser.Application, org, space string) (pl Plan)
}

type appLintPlan struct {
}

func (p appLintPlan) Plan(manifest manifestparser.Application, org, space string) (pl Plan) {
	desc := "Linting application"
	pl = append(pl, NewClientCommand(p.createFunc(manifest, org, space), desc))
	return
}

func (p appLintPlan) getMetadataInOrgSpace(client *cfclient.Client, orgName, spaceName string) (metadata cfclient.Metadata, err error) {
	org, err := client.GetOrgByName(orgName)
	if err != nil {
		return
	}
	space, err := client.GetSpaceByName(spaceName, org.Guid)
	if err != nil {
		return
	}
	meta, err := client.SpaceMetadata(space.Guid)
	if err != nil {
		return
	}
	metadata = *meta

	return
}

func (p appLintPlan) getLabelsForApp(manifest manifestparser.Application) map[any]any {
	labels := make(map[any]any)
	if manifest.RemainingManifestFields["metadata"] != nil {
		metadata := manifest.RemainingManifestFields["metadata"].(map[any]any)
		if metadata["labels"] != nil {
			labels = metadata["labels"].(map[any]any)
		}
	}
	return labels
}

func (p appLintPlan) checkProduct() {

}
func (p appLintPlan) createFunc(manifest manifestparser.Application, org, space string) func(*cfclient.Client, *logger.CapturingWriter) error {
	return func(cfClient *cfclient.Client, logger *logger.CapturingWriter) error {
		
		if manifest.Stack == "cflinuxfs3" {
			logger.Println("'stack: cflinuxfs3' is deprecated. Please update to 'cflinuxfs4'.")
		}

		labels := p.getLabelsForApp(manifest)
		manifestProduct, manifestProductFound := labels["product"]
		manifestEnvironment, manifestEnvironmentFound := labels["environment"]

		if manifestProductFound && manifestEnvironmentFound {
			logger.Println(fmt.Sprintf(`Found product '%s' and environment '%s' in manifest`, manifestProduct, manifestEnvironment))
			return nil
		}

		logger.Println(fmt.Sprintf("Fetching metadata labels set on '%s/%s'", org, space))
		metadata, err := p.getMetadataInOrgSpace(cfClient, org, space)
		if err != nil {
			logger.Println(fmt.Sprintf(`\t Failed to fetch: %s`, err.Error()))
			logger.Println("\t Lets continue...")
			return nil
		}

		spaceProduct, spaceProductFound := metadata.Labels["product"]
		spaceEnvironment, spaceEnvironmentFound := metadata.Labels["environment"]

		if manifestProductFound || spaceProductFound {
			p := manifestProduct
			in := "manifest"
			if !manifestProductFound {
				p = spaceProduct
				in = "space"
			}
			logger.Println(fmt.Sprintf("'product' is set. Found '%s' in %s", p, in))
		} else {
			logger.Println("'product' is missing in both manifest and space")
		}

		if manifestEnvironmentFound || spaceEnvironmentFound {
			e := manifestEnvironment
			in := "manifest"
			if !manifestEnvironmentFound {
				e = spaceEnvironment
				in = "space"
			}
			logger.Println(fmt.Sprintf("'environment' is set. Found '%s' in %s", e, in))
		} else {
			logger.Println("'environment' is missing in both manifest and space")
		}

		if !(manifestProductFound || spaceProductFound) || !(manifestEnvironmentFound || spaceEnvironmentFound) {
			logger.Println("Please see https://ee.public.springernature.app/inventory/ for more information about labels and how to set them!")
		}
		return nil
	}
}

func NewCheckLabelsPlan() AppLintPlan {
	return appLintPlan{}
}
