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
			Buckets: generateBuckets(80, 0, 4),
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
			Buckets: generateBuckets(80, 0, 4),
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

func generateBuckets(amount_buckets int, min float64, max float64) []float64 {
	buckets := make([]float64, amount_buckets)
	step := (max - min) / float64(amount_buckets)
	for i := range amount_buckets {
		buckets[i] = min + step*float64(i)
	}
	return buckets
}
