package monitoring

import (
	"net/http"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

var (
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, 2.0, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 2.9, 3.0, 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 3.8, 3.9, 4.0},
		},
		[]string{"handlerFunction"},
	)

	requestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "request_total",
			Help: "Total number of HTTP requests processed",
		},
		[]string{"handlerFunction"},
	)

	goHeapAlloc = promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "go_heap_alloc_bytes",
			Help: "Number of heap bytes allocated and still in use by Go",
		},
		func() float64 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return float64(m.HeapAlloc)
		},
	)

	StreamDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "kebeng",
		Subsystem: "store",
		Name:      "grpc_stream_duration_seconds",
		Help:      "Time taken for each gRPC streaming RPC",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method"})

	monitoringRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds_monitoring",
			Help:    "Duration of HTTP requests measured in monitoring service in seconds",
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, 2.0, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 2.9, 3.0, 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 3.8, 3.9, 4.0},
		},
		[]string{"handlerFunction"},
	)
)

func init() {
	prometheus.MustRegister(requestDuration, requestCount, StreamDuration, monitoringRequestDuration)
	// goHeapAlloc is already registered via promauto
}

func CreateMetricsEndpoint() {
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":9100", nil)
}

func StartTimer(handler string) func() {
	start := time.Now()
	return func() {
		requestDuration.WithLabelValues(handler).
			Observe(float64(time.Since(start).Seconds()))
		requestCount.WithLabelValues(handler).Inc()
	}
}

func StartMonitoringTimer(handler string) func() {
	start := time.Now()
	return func() {
		monitoringRequestDuration.WithLabelValues(handler).
			Observe(float64(time.Since(start).Seconds()))
	}
}

func StreamingInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	err := handler(srv, ss)
	StreamDuration.WithLabelValues(info.FullMethod).
		Observe(time.Since(start).Seconds())
	return err
}
