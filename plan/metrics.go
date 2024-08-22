package plan

import (
	"time"

	"github.com/prometheus/common/expfmt"

	"github.com/springernature/halfpipe-deploy-resource/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

type Metrics interface {
	Success() error
	Failure() error
}

func NewMetrics(request config.Request, url string) Metrics {
	labels := prometheus.Labels{
		"command": request.Params.Command,
	}
	return &prometheusMetrics{
		url:       url,
		request:   request,
		startTime: time.Now(),
		successCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        "halfpipe_cf_success",
			Help:        "Successful invocation of halfpipe cf deployment",
			ConstLabels: labels,
		}),
		failureCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        "halfpipe_cf_failure",
			Help:        "Unsuccessful invocation of halfpipe cf deployment",
			ConstLabels: labels,
		}),
		timerHistogram: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:        "halfpipe_cf_duration_seconds",
			Help:        "Time taken in seconds for successful invocation of halfpipe cf deployment",
			Buckets:     []float64{30, 60, 90, 120, 150, 180, 210, 240, 270, 300},
			ConstLabels: labels,
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
	pusher.Format(expfmt.NewFormat(expfmt.TypeTextPlain))
	for _, m := range metrics {
		pusher.Collector(m)
	}
	return pusher.Add()
}
