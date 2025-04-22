package monitoring

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func CreateMetricsEndpoint() {
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":9100", nil)
}
