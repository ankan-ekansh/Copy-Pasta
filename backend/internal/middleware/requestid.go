package middleware

import (
	"context"
	"net/http"
	"regexp"

	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-ID"

type requestIDKeyType struct{}

var requestIDCtxKey = requestIDKeyType{}

// validRequestID allows 1-64 alphanumeric + hyphen characters.
var validRequestID = regexp.MustCompile(`^[a-zA-Z0-9\-]{1,64}$`)

// RequestID middleware ensures every request has a unique request ID.
// If the incoming request has a valid X-Request-ID header, it is reused.
// Otherwise, a new UUID is generated.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if !validRequestID.MatchString(id) {
			id = uuid.New().String()
		}

		w.Header().Set(requestIDHeader, id)
		ctx := context.WithValue(r.Context(), requestIDCtxKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDCtxKey).(string); ok {
		return v
	}
	return ""
}
