// Example Prometheus metrics middleware for natsmicromw

package middleware

import (
	"time"

	"github.com/nats-io/nats.go/micro"

	// For prometheus metrics
	"github.com/prometheus/client_golang/prometheus"
)

// Define new metrics for the middleware
var (
	prometheusMessageCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nats_messages_total",
			Help: "Total number of NATS messages received.",
		},
		[]string{"subject"})

	prometheusRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nats_request_duration_seconds",
			Help:    "Duration of NATS requests in seconds.",
			Buckets: []float64{0.001, .005, .01, .025, .05, .075, .1, .25, .5, .75, 1.0, 2.5, 5.0, 7.5, 10.0},
		}, []string{"subject"})

	prometheusPayloadSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nats_payload_size_bytes",
			Help:    "Size of NATS payloads in bytes.",
			Buckets: []float64{128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536, 131072, 262144, 524288, 1048576},
			// In case bigger sizes are needed, likely not
			// , 2097152, 4194304, 8388608, 16777216, 33554432, 67108864, 134217728, 268435456, 536870912, 1073741824, 2147483648
		}, []string{"subject"})
)

func init() {
	// Register the metric with Prometheus
	prometheus.MustRegister(prometheusMessageCount)
	prometheus.MustRegister(prometheusRequestDuration)
	prometheus.MustRegister(prometheusPayloadSize)
}

// Middleware that increments the message count metric
func MetricsMiddleware(next micro.Handler) micro.Handler {
	return micro.HandlerFunc(func(req micro.Request) {
		// Increment the message count for the subject
		prometheusMessageCount.With(prometheus.Labels{"subject": req.Subject()}).Inc()

		// Record start time
		start := time.Now()

		// Call the next middleware or handler function
		next.Handle(req)

		// Record elapsed time and payload size
		elapsed := time.Since(start)
		payloadSize := len(req.Data())

		// Report metrics to Prometheus or other monitoring system
		prometheusRequestDuration.
			With(prometheus.Labels{"subject": req.Subject()}).
			Observe(float64(elapsed.Seconds()))
		prometheusPayloadSize.
			With(prometheus.Labels{"subject": req.Subject()}).
			Observe(float64(payloadSize))
	})
}
