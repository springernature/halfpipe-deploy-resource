package plan

import (
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/springernature/halfpipe-deploy-resource"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPromoteWorkerApp(t *testing.T) {
	t.Run("No previously deployed version", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "STARTED",
			},
		}

		manifest := `applications:
- name: myApp
  no-route: true
`
		man := halfpipe_deploy_resource.ParseManifest(manifest).Applications[0]

		expectedPlan := Plan{
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan([]cfclient.Domain{}).Plan(man, validRequest, summary)
		assert.Equal(t, expectedPlan, plan)
	})

	t.Run("One previously deployed stopped version", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "STARTED",
			},
			{
				Name:  "myApp",
				State: "stopped",
			},
		}

		manifest := `applications:
- name: myApp
  no-route: true
`
		man := halfpipe_deploy_resource.ParseManifest(manifest).Applications[0]

		expectedPlan := Plan{
			NewCfCommand("rename", man.Name, createOldAppName(man.Name)),
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan([]cfclient.Domain{}).Plan(man, validRequest, summary)
		//assert.Nil(t, err)
		assert.Equal(t, expectedPlan, plan)
	})

	t.Run("One previously deployed started version", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "STARTED",
			},
			{
				Name:  "myApp",
				State: "STARTED",
			},
		}

		manifest := `applications:
- name: myApp
  no-route: true
`
		man := halfpipe_deploy_resource.ParseManifest(manifest).Applications[0]

		expectedPlan := Plan{
			NewCfCommand("rename", man.Name, createOldAppName(man.Name)),
			NewCfCommand("stop", createOldAppName(man.Name)),
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan([]cfclient.Domain{}).Plan(man, validRequest, summary)
		assert.Equal(t, expectedPlan, plan)
	})

	t.Run("One previously deployed started version with an stopped old version", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "STARTED",
			},
			{
				Name:  "myApp",
				State: "STARTED",
			},
			{
				Name:  "myApp-OLD",
				State: "stopped",
			},
		}

		manifest := `applications:
- name: myApp
  no-route: true
`
		man := halfpipe_deploy_resource.ParseManifest(manifest).Applications[0]

		expectedPlan := Plan{
			NewCfCommand("rename", createOldAppName(man.Name), createDeleteName(man.Name, 0)),
			NewCfCommand("rename", man.Name, createOldAppName(man.Name)),
			NewCfCommand("stop", createOldAppName(man.Name)),
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan([]cfclient.Domain{}).Plan(man, validRequest, summary)
		assert.Equal(t, expectedPlan, plan)
	})

	t.Run("One previously deployed started version with an stopped old version and a uncleaned up DELETE app", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "STARTED",
			},
			{
				Name:  "myApp",
				State: "STARTED",
			},
			{
				Name:  "myApp-OLD",
				State: "stopped",
			},
			{
				Name:  "myApp-DELETE",
				State: "stopped",
			},
		}

		manifest := `applications:
- name: myApp
  no-route: true
`
		man := halfpipe_deploy_resource.ParseManifest(manifest).Applications[0]

		expectedPlan := Plan{
			NewCfCommand("rename", createOldAppName(man.Name), createDeleteName(man.Name, 1)),
			NewCfCommand("rename", man.Name, createOldAppName(man.Name)),
			NewCfCommand("stop", createOldAppName(man.Name)),
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan([]cfclient.Domain{}).Plan(man, validRequest, summary)
		assert.Equal(t, expectedPlan, plan)
	})

	t.Run("One previously deployed started version with an stopped old version and a couple of uncleaned DELETE apps", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "STARTED",
			},
			{
				Name:  "myApp",
				State: "STARTED",
			},
			{
				Name:  "myApp-OLD",
				State: "stopped",
			},
			{
				Name:  "myApp-DELETE",
				State: "stopped",
			},
			{
				Name:  "myApp-DELETE-1",
				State: "stopped",
			},
			{
				Name:  "myApp-DELETE-2",
				State: "stopped",
			},
		}

		manifest := `applications:
- name: myApp
  no-route: true
`
		man := halfpipe_deploy_resource.ParseManifest(manifest).Applications[0]

		expectedPlan := Plan{
			NewCfCommand("rename", createOldAppName(man.Name), createDeleteName(man.Name, 3)),
			NewCfCommand("rename", man.Name, createOldAppName(man.Name)),
			NewCfCommand("stop", createOldAppName(man.Name)),
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan([]cfclient.Domain{}).Plan(man, validRequest, summary)
		assert.Equal(t, expectedPlan, plan)
	})
}

func TestPromoteNormalApp(t *testing.T) {
	t.Run("No previously deployed version and routes in the manifest", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "STARTED",
			},
		}

		manifest := `applications:
- name: myApp
  routes:
  - route: myroute.domain1.com
  - route: myroute.subroute.domain2.com
`
		man := halfpipe_deploy_resource.ParseManifest(manifest).Applications[0]

		expectedPlan := Plan{
			NewCfCommand("map-route", createCandidateAppName(man.Name), "domain1.com", "--hostname", "myroute"),
			NewCfCommand("map-route", createCandidateAppName(man.Name), "subroute.domain2.com", "--hostname", "myroute"),
			NewCfCommand("unmap-route", createCandidateAppName(man.Name), validRequest.Params.TestDomain, "--hostname", createCandidateHostname(man, validRequest)),
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan([]cfclient.Domain{}).Plan(man, validRequest, summary)
		assert.Equal(t, expectedPlan, plan)
	})

	t.Run("No previously deployed version and routes in the manifest with path", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "STARTED",
			},
		}

		manifest := `applications:
- name: myApp
  routes:
  - route: myroute.domain1.com
  - route: myroute.subroute.domain2.com/pathy/path
`
		man := halfpipe_deploy_resource.ParseManifest(manifest).Applications[0]

		expectedPlan := Plan{
			NewCfCommand("map-route", createCandidateAppName(man.Name), "domain1.com", "--hostname", "myroute"),
			NewCfCommand("map-route", createCandidateAppName(man.Name), "subroute.domain2.com", "--hostname", "myroute", "--path", "pathy/path"),
			NewCfCommand("unmap-route", createCandidateAppName(man.Name), validRequest.Params.TestDomain, "--hostname", createCandidateHostname(man, validRequest)),
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan([]cfclient.Domain{}).Plan(man, validRequest, summary)
		assert.Equal(t, expectedPlan, plan)
	})

	t.Run("No previously deployed version and a route that is a domain", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "STARTED",
			},
		}

		manifest := `applications:
- name: myApp
  routes:
  - route: myroute.domain1.com
  - route: thisIsASpaceOwnedDomain.com
`
		man := halfpipe_deploy_resource.ParseManifest(manifest).Applications[0]

		privateRoutesInOrg := []cfclient.Domain{
			{
				Name: "thisIsASpaceOwnedDomain.com",
			},
		}

		expectedPlan := Plan{
			NewCfCommand("map-route", createCandidateAppName(man.Name), "domain1.com", "--hostname", "myroute"),
			NewCfCommand("map-route", createCandidateAppName(man.Name), "thisIsASpaceOwnedDomain.com"),
			NewCfCommand("unmap-route", createCandidateAppName(man.Name), validRequest.Params.TestDomain, "--hostname", createCandidateHostname(man, validRequest)),
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan(privateRoutesInOrg).Plan(man, validRequest, summary)
		assert.Equal(t, expectedPlan, plan)
	})

	t.Run("No previously deployed version and a route that is a domain with a path", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "STARTED",
			},
		}

		manifest := `applications:
- name: myApp
  routes:
  - route: myroute.domain1.com
  - route: thisIsASpaceOwnedDomain.com/mypath
`
		man := halfpipe_deploy_resource.ParseManifest(manifest).Applications[0]

		privateRoutesInOrg := []cfclient.Domain{
			{
				Name: "thisIsASpaceOwnedDomain.com",
			},
		}

		expectedPlan := Plan{
			NewCfCommand("map-route", createCandidateAppName(man.Name), "domain1.com", "--hostname", "myroute"),
			NewCfCommand("map-route", createCandidateAppName(man.Name), "thisIsASpaceOwnedDomain.com", "--path", "mypath"),
			NewCfCommand("unmap-route", createCandidateAppName(man.Name), validRequest.Params.TestDomain, "--hostname", createCandidateHostname(man, validRequest)),
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan(privateRoutesInOrg).Plan(man, validRequest, summary)
		assert.Equal(t, expectedPlan, plan)
	})

	t.Run("No previously deployed version and a route that is a sub domain", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "STARTED",
			},
		}

		manifest := `applications:
- name: myApp
  routes:
  - route: myroute.domain1.com
  - route: subroute.thisIsASpaceOwnedDomain.com
`
		man := halfpipe_deploy_resource.ParseManifest(manifest).Applications[0]

		privateRoutesInOrg := []cfclient.Domain{
			{
				Name: "thisIsASpaceOwnedDomain.com",
			},
		}

		expectedPlan := Plan{
			NewCfCommand("map-route", createCandidateAppName(man.Name), "domain1.com", "--hostname", "myroute"),
			NewCfCommand("map-route", createCandidateAppName(man.Name), "thisIsASpaceOwnedDomain.com", "--hostname", "subroute"),
			NewCfCommand("unmap-route", createCandidateAppName(man.Name), validRequest.Params.TestDomain, "--hostname", createCandidateHostname(man, validRequest)),
			NewCfCommand("rename", createCandidateAppName(man.Name), man.Name),
		}

		plan := NewPromotePlan(privateRoutesInOrg).Plan(man, validRequest, summary)
		assert.Equal(t, expectedPlan, plan)
	})

	t.Run("If there is already a DELETE-1", func(t *testing.T) {
		summary := []cfclient.AppSummary{
			{
				Name:  "myApp-CANDIDATE",
				State: "STARTED",
			},
			{
				Name: "myApp",
			},
			{
				Name: "myApp-OLD",
			},
			{
				Name: "myApp-DELETE-1",
			},
			{
				Name: "myApp-DELETE-2",
			},
		}

		manifest := `applications:
- name: myApp
  routes:
  - route: myroute.domain1.com
  - route: subroute.thisIsASpaceOwnedDomain.com
`
		man := halfpipe_deploy_resource.ParseManifest(manifest).Applications[0]

		privateRoutesInOrg := []cfclient.Domain{
			{
				Name: "thisIsASpaceOwnedDomain.com",
			},
		}

		expectedPlan := Plan{
			NewCfCommand("map-route", "myApp-CANDIDATE", "domain1.com", "--hostname", "myroute"),
			NewCfCommand("map-route", "myApp-CANDIDATE", "thisIsASpaceOwnedDomain.com", "--hostname", "subroute"),
			NewCfCommand("unmap-route", "myApp-CANDIDATE", validRequest.Params.TestDomain, "--hostname", createCandidateHostname(man, validRequest)),
			NewCfCommand("rename", "myApp-OLD", "myApp-DELETE"),
			NewCfCommand("rename", "myApp", "myApp-OLD"),
			NewCfCommand("rename", "myApp-CANDIDATE", "myApp"),
		}

		plan := NewPromotePlan(privateRoutesInOrg).Plan(man, validRequest, summary)
		assert.Equal(t, expectedPlan, plan)
	})
}
