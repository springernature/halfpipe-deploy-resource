package plan

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/springernature/halfpipe-deploy-resource/config"

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
	assert.True(t, strings.HasPrefix(path, "/metrics/job/promote"), path)

	err = m.Failure()
	assert.Nil(t, err)
	assert.Equal(t, 2, counter)
}
