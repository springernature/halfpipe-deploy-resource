package plan

import (
	"github.com/springernature/halfpipe-deploy-resource/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"strings"

	"github.com/stretchr/testify/assert"
)

func TestNewPrometheusMetrics(t *testing.T) {
	var path string
	var counter int
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		counter++
		w.WriteHeader(202)
	}))
	defer gateway.Close()

	m := NewMetrics(config.Request{
		Source: config.Source{
			API:   "some/cf.api",
			Org:   "some-cf-org",
			Space: "some-cf-space",
		},
		Params: config.Params{
			Command: "promote",
		},
		Metadata: config.Metadata{
			AppName: "some-App-name",
		},
	}, gateway.URL)

	err := m.Success()
	assert.Nil(t, err)
	assert.Equal(t, 1, counter)
	assert.True(t, strings.HasPrefix(path, "/metrics/job/promote/"), path)
	assert.Contains(t, path, "cf_api/some_cf_api")
	assert.Contains(t, path, "cf_org/some_cf_org")
	assert.Contains(t, path, "cf_space/some_cf_space")
	assert.Contains(t, path, "app_name/some_App_name")


	err = m.Failure()
	assert.Nil(t, err)
	assert.Equal(t, 2, counter)
}
