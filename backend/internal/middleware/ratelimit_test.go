package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitConvert_UnderLimit(t *testing.T) {
	handler := RateLimitConvert()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request should pass
	req := httptest.NewRequest("POST", "/api/convert", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestRateLimitConvert_OverLimit(t *testing.T) {
	handler := RateLimitConvert()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust the default limit (10 requests)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/api/convert", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// 11th request should be rate limited
	req := httptest.NewRequest("POST", "/api/convert", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}

	// Verify JSON response body
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["error"] != "rate limit exceeded, try again later" {
		t.Errorf("unexpected error message: %q", body["error"])
	}

	// Verify Retry-After header
	if rec.Header().Get("Retry-After") != "60" {
		t.Errorf("expected Retry-After: 60, got %q", rec.Header().Get("Retry-After"))
	}
}

func TestRateLimitConvert_DifferentIPsAreIndependent(t *testing.T) {
	handler := RateLimitConvert()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust limit for IP A
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/api/convert", nil)
		req.RemoteAddr = "10.0.0.2:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// IP B should still work
	req := httptest.NewRequest("POST", "/api/convert", nil)
	req.RemoteAddr = "10.0.0.3:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("different IP should not be limited, got %d", rec.Code)
	}
}

func TestEnvIntOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		defVal   int
		expected int
	}{
		{"empty uses default", "", 10, 10},
		{"valid int", "20", 10, 20},
		{"invalid string", "abc", 10, 10},
		{"zero uses default", "0", 10, 10},
		{"negative uses default", "-5", 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_RATE_LIMIT_" + tt.name
			if tt.envVal != "" {
				t.Setenv(key, tt.envVal)
			}
			got := envIntOrDefault(key, tt.defVal)
			if got != tt.expected {
				t.Errorf("envIntOrDefault(%q, %d) = %d, want %d", key, tt.defVal, got, tt.expected)
			}
		})
	}
}
