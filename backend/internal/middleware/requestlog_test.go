package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRequestLog_CapturesStatusAndBytes(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	slog.SetDefault(logger)

	// Set up chi router so RouteContext is available
	r := chi.NewRouter()
	r.Use(RequestID)
	r.Use(RequestLog)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("hello"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	// Parse logged JSON
	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log entry: %v\nraw: %s", err, buf.String())
	}

	// Check fields
	if entry["method"] != "GET" {
		t.Errorf("expected method GET, got %v", entry["method"])
	}
	if entry["status"] != float64(201) {
		t.Errorf("expected status 201, got %v", entry["status"])
	}
	if entry["bytes"] != float64(5) {
		t.Errorf("expected bytes 5, got %v", entry["bytes"])
	}
	if entry["request_id"] == nil || entry["request_id"] == "" {
		t.Error("expected request_id in log")
	}
	if entry["route"] != "/test" {
		t.Errorf("expected route /test, got %v", entry["route"])
	}
	if _, ok := entry["latency_ms"]; !ok {
		t.Error("expected latency_ms in log")
	}
}

func TestRequestLog_IncludesRequestIDFromContext(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	slog.SetDefault(logger)

	customID := "my-custom-id-123"

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), requestIDCtxKey, customID)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Use(RequestLog)
	r.Get("/foo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/foo", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log: %v", err)
	}

	if entry["request_id"] != customID {
		t.Errorf("expected request_id %q, got %v", customID, entry["request_id"])
	}
}
