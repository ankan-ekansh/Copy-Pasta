package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const sessionIDKey contextKey = "sessionID"

const cookieName = "copy-pasta-session"

// Session is middleware that ensures every request has a session ID.
// If the cookie is missing, a new UUID is generated and set.
// Skips paths that don't need sessions (e.g., /metrics for Prometheus scrapes).
func Session(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip session for internal/infra endpoints
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		var sessionID string

		cookie, err := r.Cookie(cookieName)
		if err == nil && cookie.Value != "" {
			sessionID = cookie.Value
		} else {
			sessionID = uuid.New().String()
			secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    sessionID,
				Path:     "/",
				MaxAge:   365 * 24 * 60 * 60, // 1 year
				HttpOnly: true,
				Secure:   secure,
				SameSite: http.SameSiteLaxMode,
			})
		}

		ctx := context.WithValue(r.Context(), sessionIDKey, sessionID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetSessionID retrieves the session ID from the request context.
func GetSessionID(ctx context.Context) string {
	if v, ok := ctx.Value(sessionIDKey).(string); ok {
		return v
	}
	return ""
}
