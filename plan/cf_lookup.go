package plan

import (
	"context"

	cfclient "github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
)

// getOrgAndSpace finds the org and space resources for the given org/space names.
func getOrgAndSpace(ctx context.Context, cf *cfclient.Client, orgName, spaceName string) (org *resource.Organization, space *resource.Space, err error) {
	orgOpts := cfclient.NewOrganizationListOptions()
	orgOpts.Names = cfclient.Filter{Values: []string{orgName}}
	org, err = cf.Organizations.Single(ctx, orgOpts)
	if err != nil {
		return
	}

	spaceOpts := cfclient.NewSpaceListOptions()
	spaceOpts.Names = cfclient.Filter{Values: []string{spaceName}}
	spaceOpts.OrganizationGUIDs = cfclient.Filter{Values: []string{org.GUID}}
	space, err = cf.Spaces.Single(ctx, spaceOpts)
	return
}

// getAppsInOrgSpace returns all apps in the given org/space.
func getAppsInOrgSpace(ctx context.Context, cf *cfclient.Client, orgName, spaceName string) (apps []*resource.App, err error) {
	_, space, err := getOrgAndSpace(ctx, cf, orgName, spaceName)
	if err != nil {
		return
	}

	appOpts := cfclient.NewAppListOptions()
	appOpts.SpaceGUIDs = cfclient.Filter{Values: []string{space.GUID}}
	apps, err = cf.Applications.ListAll(ctx, appOpts)
	return
}
