package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "Duration of HTTP requests in milliseconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2, 5},
		},
		[]string{"handlerFunction"},
	)
)

func init() {
	prometheus.MustRegister(requestDuration)
}

func StartTimer(handler string) func() {
	start := time.Now()
	return func() {
		requestDuration.WithLabelValues(handler).
			Observe(float64(time.Since(start).Milliseconds()))
	}
}
