package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "request_duration_seconds",
			Help: "Duration of HTTP requests in seconds",
			Buckets: []float64{
				0.05, 0.1, 0.15, 0.2, 0.25, 0.3,
				0.35, 0.4, 0.45, 0.5, 0.55,
				0.6, 0.65, 0.7, 0.75, 0.8,
				0.85, 0.9, 0.95, 1.0, 1.1, 1.2, 1.3, 1.4, 1.5,
			},
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
			Observe(float64(time.Since(start).Seconds()))
	}
}
