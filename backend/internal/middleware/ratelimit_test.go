package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestRateLimitConvert_UnderLimit(t *testing.T) {
	t.Setenv("RATE_LIMIT_CONVERT", "10")
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
	t.Setenv("RATE_LIMIT_CONVERT", "10")
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

	// Verify Retry-After header is derived from rateLimitWindow
	expectedRetryAfter := strconv.Itoa(int(rateLimitWindow.Seconds()))
	if rec.Header().Get("Retry-After") != expectedRetryAfter {
		t.Errorf("expected Retry-After: %s, got %q", expectedRetryAfter, rec.Header().Get("Retry-After"))
	}
}

func TestRateLimitConvert_DifferentIPsAreIndependent(t *testing.T) {
	t.Setenv("RATE_LIMIT_CONVERT", "10")
	handler := RateLimitConvert()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust limit for IP A and verify each succeeds
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/api/convert", nil)
		req.RemoteAddr = "10.0.0.2:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("IP A request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// Verify IP A is now rate-limited
	req := httptest.NewRequest("POST", "/api/convert", nil)
	req.RemoteAddr = "10.0.0.2:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("IP A should be rate-limited, got %d", rec.Code)
	}

	// IP B should still work
	req = httptest.NewRequest("POST", "/api/convert", nil)
	req.RemoteAddr = "10.0.0.3:12345"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("IP B should not be limited, got %d", rec.Code)
	}
}

func TestEnvIntOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envVal   string
		setEnv   bool
		defVal   int
		expected int
	}{
		{"empty uses default", "TEST_RL_EMPTY", "", true, 10, 10},
		{"valid int", "TEST_RL_VALID", "20", true, 10, 20},
		{"invalid string", "TEST_RL_INVALID", "abc", true, 10, 10},
		{"zero uses default", "TEST_RL_ZERO", "0", true, 10, 10},
		{"negative uses default", "TEST_RL_NEGATIVE", "-5", true, 10, 10},
		{"unset uses default", "TEST_RL_UNSET", "", false, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(tt.key, tt.envVal)
			}
			got := envIntOrDefault(tt.key, tt.defVal)
			if got != tt.expected {
				t.Errorf("envIntOrDefault(%q, %d) = %d, want %d", tt.key, tt.defVal, got, tt.expected)
			}
		})
	}
}

func TestRateLimitAPI_EnforcesLimit(t *testing.T) {
	t.Setenv("RATE_LIMIT_API", "5")
	handler := RateLimitAPI()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust the configured limit (5 requests)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/api/pastas", nil)
		req.RemoteAddr = "10.0.0.50:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// 6th request should be rate limited
	req := httptest.NewRequest("GET", "/api/pastas", nil)
	req.RemoteAddr = "10.0.0.50:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}

	// Verify same 429 response format as RateLimitConvert
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["error"] != "rate limit exceeded, try again later" {
		t.Errorf("unexpected error message: %q", body["error"])
	}

	expectedRetryAfter := strconv.Itoa(int(rateLimitWindow.Seconds()))
	if rec.Header().Get("Retry-After") != expectedRetryAfter {
		t.Errorf("expected Retry-After: %s, got %q", expectedRetryAfter, rec.Header().Get("Retry-After"))
	}
}
