package plan

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
	"context"
	"fmt"
	cfclient "github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"github.com/gookit/color"
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

func (p appLintPlan) getMetadataInOrgSpace(ctx context.Context, cf *cfclient.Client, orgName, spaceName string) (metadata resource.Metadata, err error) {
	_, space, err := getOrgAndSpace(ctx, cf, orgName, spaceName)
	if err != nil {
		return
	}
	if space.Metadata != nil {
		metadata = *space.Metadata
	}
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

func stringLabel(labels map[string]*string, key string) (value string, found bool) {
	v, ok := labels[key]
	if !ok || v == nil {
		return "", false
	}
	return *v, true
}

func (p appLintPlan) createFunc(manifest manifestparser.Application, org, space string) func(*cfclient.Client, *logger.CapturingWriter) error {
	return func(cfClient *cfclient.Client, logger *logger.CapturingWriter) error {

		printWarning := func(msg string) {
			logger.Println(color.New(color.FgRed).Sprintf("**WARNING** "), msg)
		}

		if manifest.Stack == "cflinuxfs3" {
			printWarning("CF stack 'cflinuxfs3' is deprecated. Please see <https://ee.public.springernature.app/paas/cf/stacks/>")
		}

		labels := p.getLabelsForApp(manifest)
		manifestProduct, manifestProductFound := labels["product"]
		manifestEnvironment, manifestEnvironmentFound := labels["environment"]
		manifestEAID, manifestEAIDFound := labels["eaid"]

		if manifestProductFound && manifestEnvironmentFound && manifestEAIDFound {
			logger.Println(fmt.Sprintf(`Found product '%s', environment '%s'  and EAID '%s' in manifest`, manifestProduct, manifestEnvironment, manifestEAID))
			return nil
		}

		logger.Println(fmt.Sprintf("Fetching metadata labels set on '%s/%s'", org, space))
		metadata, err := p.getMetadataInOrgSpace(context.Background(), cfClient, org, space)
		if err != nil {
			logger.Println(fmt.Sprintf(`\t Failed to fetch: %s`, err.Error()))
			logger.Println("\t Lets continue...")
			return nil
		}

		spaceProduct, spaceProductFound := stringLabel(metadata.Labels, "product")
		spaceEnvironment, spaceEnvironmentFound := stringLabel(metadata.Labels, "environment")
		spaceEAID, spaceEAIDFound := stringLabel(metadata.Labels, "eaid")

		if manifestProductFound || spaceProductFound {
			p := manifestProduct
			in := "manifest"
			if !manifestProductFound {
				p = spaceProduct
				in = "space"
			}
			logger.Println(fmt.Sprintf("'product' is set. Found '%s' in %s", p, in))
		} else {
			printWarning("'product' is missing in both manifest and space")
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
			printWarning("'environment' is missing in both manifest and space")
		}

		if manifestEAIDFound || spaceEAIDFound {
			e := manifestEAID
			in := "manifest"
			if !manifestEAIDFound {
				e = spaceEAID
				in = "space"
			}
			logger.Println(fmt.Sprintf("'eaid' is set. Found '%s' in %s", e, in))
		} else {
			printWarning("'eaid' is missing in both manifest and space")
		}

		if !(manifestProductFound || spaceProductFound || spaceEAIDFound) || !(manifestEnvironmentFound || spaceEnvironmentFound || manifestEAIDFound) {
			logger.Println("Please see https://ee.public.springernature.app/inventory/ for more information about labels and how to set them!")
		}
		return nil
	}
}

func NewCheckLabelsPlan() AppLintPlan {
	return appLintPlan{}
}
