package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/metrics"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// Metrics records HTTP request metrics (count, duration, response size) for Prometheus.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		route := "unmatched"
		if rctx := chi.RouteContext(r.Context()); rctx != nil {
			if pattern := rctx.RoutePattern(); pattern != "" {
				route = pattern
			}
		}

		status := strconv.Itoa(ww.Status())
		duration := time.Since(start).Seconds()

		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, route, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, route).Observe(duration)
		metrics.HTTPResponseSize.WithLabelValues(r.Method, route).Observe(float64(ww.BytesWritten()))
	})
}
