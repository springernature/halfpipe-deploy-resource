package plan

import (
	"github.com/prometheus/common/expfmt"
	"time"

	"github.com/springernature/halfpipe-deploy-resource/config"

	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

type Metrics interface {
	Success() error
	Failure() error
}

func NewMetrics(request config.Request, url string) Metrics {
	return &prometheusMetrics{
		url:       url,
		request:   request,
		startTime: time.Now(),
		successCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "halfpipe_cf_success",
			Help: "Successful invocation of halfpipe cf deployment",
		}),
		failureCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "halfpipe_cf_failure",
			Help: "Unsuccessful invocation of halfpipe cf deployment",
		}),
		timerHistogram: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "halfpipe_cf_duration_seconds",
			Help:    "Time taken in seconds for successful invocation of halfpipe cf deployment",
			Buckets: []float64{30, 60, 90, 120, 150, 180, 210, 240, 270, 300},
		}),
	}
}

type prometheusMetrics struct {
	url            string
	request        config.Request
	startTime      time.Time
	successCounter prometheus.Counter
	failureCounter prometheus.Counter
	timerHistogram prometheus.Histogram
}

func (p *prometheusMetrics) Success() error {
	p.successCounter.Inc()
	p.timerHistogram.Observe(time.Now().Sub(p.startTime).Seconds())
	return p.push(p.successCounter, p.timerHistogram)
}

func (p *prometheusMetrics) Failure() error {
	p.failureCounter.Inc()
	return p.push(p.failureCounter)
}

func (p *prometheusMetrics) push(metrics ...prometheus.Collector) error {
	pusher := push.New(p.url, p.request.Params.Command)
	pusher.Format(expfmt.FmtText)
	pusher.Grouping("cf_api", sanitize(p.request.Source.API))
	for _, m := range metrics {
		pusher.Collector(m)
	}
	return pusher.Add()
}

func sanitize(s string) string {
	return regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(s, "_")
}
