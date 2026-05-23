package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// RequestLog is a structured access log middleware using slog.
// It logs method, route pattern, status, latency, response size, and request ID.
func RequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		// Use Chi's route pattern to avoid unbounded label cardinality
		routePattern := "unmatched"
		if rctx := chi.RouteContext(r.Context()); rctx != nil {
			if pattern := rctx.RoutePattern(); pattern != "" {
				routePattern = pattern
			}
		}

		attrs := []any{
			"request_id", GetRequestID(r.Context()),
			"method", r.Method,
			"route", routePattern,
			"status", ww.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"bytes", ww.BytesWritten(),
			"remote_addr", r.RemoteAddr,
		}
		// Only include raw path for unmatched routes (debugging unknown endpoints)
		if routePattern == "unmatched" {
			attrs = append(attrs, "path", r.URL.Path)
		}

		slog.Info("request completed", attrs...)
	})
}
