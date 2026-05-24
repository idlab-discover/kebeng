package monitoring

import (
	"context"
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
		Buckets: []float64{
			0.0001, 0.00025, 0.0005, 0.00075,
			0.001, 0.0025, 0.005, 0.0075,
			0.01, 0.025, 0.05, 0.075,
			0.1, 0.25, 0.5, 0.75,
			1.0, 2.5, 5.0, 10.0,
		},
	}, []string{"method"})

	UnaryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "kebeng",
		Subsystem: "store",
		Name:      "grpc_unary_duration_seconds",
		Help:      "Time taken for each gRPC unary RPC",
		Buckets: []float64{
			0.0001, 0.00025, 0.0005, 0.00075,
			0.001, 0.0025, 0.005, 0.0075,
			0.01, 0.025, 0.05, 0.075,
			0.1, 0.25, 0.5, 0.75,
			1.0, 2.5, 5.0, 10.0,
		},
	}, []string{"method"})

	monitoringRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds_monitoring",
			Help:    "Duration of HTTP requests measured in monitoring service in seconds",
			Buckets: generateBuckets(250, 0, 3),
		},
		[]string{"handlerFunction"},
	)
)

func init() {
	prometheus.MustRegister(requestDuration, requestCount, StreamDuration, UnaryDuration, monitoringRequestDuration)
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
		dur := time.Since(start)
		monitoringRequestDuration.WithLabelValues(handler).
			Observe(float64(dur.Seconds()))

		RecordToFile(handler, dur)
	}
}

func StreamingInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	err := handler(srv, ss)
	StreamDuration.WithLabelValues(info.FullMethod).
		Observe(time.Since(start).Seconds())
	return err
}

func UnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	UnaryDuration.WithLabelValues(info.FullMethod).
		Observe(time.Since(start).Seconds())
	return resp, err
}

func generateBuckets(amount_buckets int, min float64, max float64) []float64 {
	buckets := make([]float64, amount_buckets)
	step := (max - min) / float64(amount_buckets)
	for i := range amount_buckets {
		buckets[i] = min + step*float64(i)
	}
	return buckets
}
