// Package metrics defines and registers Prometheus metrics for the application.
package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// HTTPRequestsTotal counts total HTTP requests by method, route, and status.
	HTTPRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "route", "status"})

	// HTTPRequestDuration tracks request latency by method and route.
	HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})

	// HTTPResponseSize tracks response body size by method and route.
	HTTPResponseSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_response_size_bytes",
		Help:    "HTTP response size in bytes.",
		Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000},
	}, []string{"method", "route"})

	// ConversionsTotal counts conversions by mode.
	ConversionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "conversions_total",
		Help: "Total number of image-to-ASCII conversions.",
	}, []string{"mode"})

	// ConversionDuration tracks conversion processing time by mode.
	ConversionDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "conversion_duration_seconds",
		Help:    "Image conversion duration in seconds.",
		Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	}, []string{"mode"})

	// DBOperationsTotal counts database operations by operation and status.
	DBOperationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "db_operations_total",
		Help: "Total number of database operations.",
	}, []string{"operation", "status"})

	// DBOperationDuration tracks database operation latency.
	DBOperationDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "db_operation_duration_seconds",
		Help:    "Database operation duration in seconds.",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	}, []string{"operation"})
)

// Register registers all application metrics with the default prometheus registry.
func Register() {
	prometheus.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		HTTPResponseSize,
		ConversionsTotal,
		ConversionDuration,
		DBOperationsTotal,
		DBOperationDuration,
	)
}
