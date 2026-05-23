package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID_GeneratesNewID(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id == "" {
			t.Error("expected non-empty request ID in context")
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	header := rec.Header().Get("X-Request-ID")
	if header == "" {
		t.Error("expected X-Request-ID response header")
	}
}

func TestRequestID_ReusesValidHeader(t *testing.T) {
	validID := "abc-123-def-456"
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id != validID {
			t.Errorf("expected request ID %q, got %q", validID, id)
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", validID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") != validID {
		t.Errorf("expected response header to contain %q", validID)
	}
}

func TestRequestID_RejectsInvalidHeader(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"too long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{"special chars", "abc<script>alert(1)</script>"},
		{"spaces", "has spaces"},
		{"empty", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id := GetRequestID(r.Context())
				if id == tc.value {
					t.Errorf("should have rejected invalid ID %q", tc.value)
				}
				if id == "" {
					t.Error("expected a generated ID, got empty")
				}
			}))

			req := httptest.NewRequest("GET", "/", nil)
			if tc.value != "" {
				req.Header.Set("X-Request-ID", tc.value)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		})
	}
}
