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

// isTrustedProxy returns true when the app is running behind a trusted
// reverse proxy that overwrites X-Real-IP / X-Forwarded-For headers.
// Set TRUSTED_PROXY=true in environments with a proxy (Docker Compose, Azure).
// When false (e.g., bare `make dev-backend`), rate limiting keys on RemoteAddr
// to prevent IP spoofing via forged headers.
func isTrustedProxy() bool {
	v, err := strconv.ParseBool(os.Getenv("TRUSTED_PROXY"))
	return err == nil && v
}

// ipKeyFunc returns the appropriate httprate key function based on
// whether the app is behind a trusted proxy.
func ipKeyFunc() httprate.Option {
	if isTrustedProxy() {
		return httprate.WithKeyByRealIP()
	}
	return httprate.WithKeyByIP()
}

// RateLimitConvert returns a rate limiter scoped to the /api/convert endpoint.
// Configurable via RATE_LIMIT_CONVERT env var (default: 10 requests/minute per IP).
func RateLimitConvert() func(http.Handler) http.Handler {
	limit := envIntOrDefault("RATE_LIMIT_CONVERT", defaultConvertLimit)
	return httprate.Limit(limit, rateLimitWindow,
		ipKeyFunc(),
		httprate.WithLimitHandler(rateLimitExceededHandler()),
	)
}

// RateLimitAPI returns a rate limiter for general API routes (excludes /api/convert,
// which has its own stricter limiter via RateLimitConvert).
// Configurable via RATE_LIMIT_API env var (default: 100 requests/minute per IP).
func RateLimitAPI() func(http.Handler) http.Handler {
	limit := envIntOrDefault("RATE_LIMIT_API", defaultAPILimit)
	return httprate.Limit(limit, rateLimitWindow,
		ipKeyFunc(),
		httprate.WithLimitHandler(rateLimitExceededHandler()),
	)
}

func rateLimitExceededHandler() http.HandlerFunc {
	retryAfter := strconv.Itoa(int(rateLimitWindow.Seconds()))
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", retryAfter)
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
