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
			// Sub-millisecond (100µs steps)
			0.0001, 0.0002, 0.0003, 0.0004, 0.0005,
			0.0006, 0.0007, 0.0008, 0.0009,

			// 1ms–10ms (500µs steps)
			0.001, 0.0015, 0.002, 0.0025, 0.003, 0.0035, 0.004, 0.0045,
			0.005, 0.0055, 0.006, 0.0065, 0.007, 0.0075, 0.008, 0.0085,
			0.009, 0.0095,

			// 10ms–100ms (2ms steps)
			0.010, 0.012, 0.014, 0.016, 0.018,
			0.020, 0.022, 0.024, 0.026, 0.028,
			0.030, 0.032, 0.034, 0.036, 0.038,
			0.040, 0.042, 0.044, 0.046, 0.048,
			0.050, 0.055, 0.060, 0.065, 0.070,
			0.075, 0.080, 0.085, 0.090, 0.095,

			// 100ms–1s (25ms steps)
			0.100, 0.125, 0.150, 0.175, 0.200,
			0.225, 0.250, 0.275, 0.300, 0.350,
			0.400, 0.450, 0.500, 0.550, 0.600,
			0.650, 0.700, 0.750, 0.800, 0.850,
			0.900, 0.950,

			// 1s–10s (250ms steps)
			1.0, 1.25, 1.5, 1.75,
			2.0, 2.25, 2.5, 2.75,
			3.0, 3.5, 4.0, 4.5,
			5.0, 5.5, 6.0, 6.5,
			7.0, 7.5, 8.0, 8.5,
			9.0, 9.5,

			// 10s–60s (1s steps)
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
			20, 22, 24, 26, 28,
			30, 35, 40, 45, 50, 55,

			// 60s–300s (10s steps)
			60, 70, 80, 90, 100, 110, 120,
			150, 180, 210, 240, 270, 300,
		},
	}, []string{"method"})

	UnaryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "kebeng",
		Subsystem: "store",
		Name:      "grpc_unary_duration_seconds",
		Help:      "Time taken for each gRPC unary RPC",
		Buckets: []float64{
			// Sub-millisecond (100µs steps)
			0.0001, 0.0002, 0.0003, 0.0004, 0.0005,
			0.0006, 0.0007, 0.0008, 0.0009,

			// 1ms–10ms (500µs steps)
			0.001, 0.0015, 0.002, 0.0025, 0.003, 0.0035, 0.004, 0.0045,
			0.005, 0.0055, 0.006, 0.0065, 0.007, 0.0075, 0.008, 0.0085,
			0.009, 0.0095,

			// 10ms–100ms (2ms steps)
			0.010, 0.012, 0.014, 0.016, 0.018,
			0.020, 0.022, 0.024, 0.026, 0.028,
			0.030, 0.032, 0.034, 0.036, 0.038,
			0.040, 0.042, 0.044, 0.046, 0.048,
			0.050, 0.055, 0.060, 0.065, 0.070,
			0.075, 0.080, 0.085, 0.090, 0.095,

			// 100ms–1s (25ms steps)
			0.100, 0.125, 0.150, 0.175, 0.200,
			0.225, 0.250, 0.275, 0.300, 0.350,
			0.400, 0.450, 0.500, 0.550, 0.600,
			0.650, 0.700, 0.750, 0.800, 0.850,
			0.900, 0.950,

			// 1s–10s (250ms steps)
			1.0, 1.25, 1.5, 1.75,
			2.0, 2.25, 2.5, 2.75,
			3.0, 3.5, 4.0, 4.5,
			5.0, 5.5, 6.0, 6.5,
			7.0, 7.5, 8.0, 8.5,
			9.0, 9.5,

			// 10s–60s (1s steps)
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
			20, 22, 24, 26, 28,
			30, 35, 40, 45, 50, 55,

			// 60s–300s (10s steps)
			60, 70, 80, 90, 100, 110, 120,
			150, 180, 210, 240, 270, 300,
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
