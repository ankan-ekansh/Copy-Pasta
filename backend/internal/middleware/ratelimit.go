package middleware

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/httprate"
)

const (
	defaultConvertLimit = 10
	defaultAPILimit     = 100
	rateLimitWindow     = 1 * time.Minute
)

// RateLimitConvert returns a rate limiter scoped to the /api/convert endpoint.
// Configurable via RATE_LIMIT_CONVERT env var (default: 10 requests/minute per IP).
func RateLimitConvert() func(http.Handler) http.Handler {
	limit := envIntOrDefault("RATE_LIMIT_CONVERT", defaultConvertLimit)
	return httprate.Limit(limit, rateLimitWindow,
		httprate.WithKeyByRealIP(),
		httprate.WithLimitHandler(rateLimitExceededHandler()),
	)
}

// RateLimitAPI returns a general rate limiter for all API endpoints.
// Configurable via RATE_LIMIT_API env var (default: 100 requests/minute per IP).
func RateLimitAPI() func(http.Handler) http.Handler {
	limit := envIntOrDefault("RATE_LIMIT_API", defaultAPILimit)
	return httprate.Limit(limit, rateLimitWindow,
		httprate.WithKeyByRealIP(),
		httprate.WithLimitHandler(rateLimitExceededHandler()),
	)
}

func rateLimitExceededHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "rate limit exceeded, try again later",
		})
	}
}

func envIntOrDefault(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultVal
}
