package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/metrics"
	"github.com/go-chi/chi/v5"
	dto "github.com/prometheus/client_model/go"
)

func TestMetrics_RecordsHTTPMetrics(t *testing.T) {
	metrics.HTTPRequestsTotal.Reset()

	r := chi.NewRouter()
	r.Use(Metrics)
	r.Get("/test-endpoint", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("hello"))
	})

	req := httptest.NewRequest("GET", "/test-endpoint", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	counter, err := metrics.HTTPRequestsTotal.GetMetricWithLabelValues("GET", "/test-endpoint", "201")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}

	m := &dto.Metric{}
	if err := counter.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}

	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected counter value 1, got %f", m.GetCounter().GetValue())
	}
}

func TestMetrics_UnmatchedRoute(t *testing.T) {
	metrics.HTTPRequestsTotal.Reset()

	r := chi.NewRouter()
	r.Use(Metrics)
	r.Get("/known", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/unknown-path", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	counter, err := metrics.HTTPRequestsTotal.GetMetricWithLabelValues("GET", "unmatched", "404")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}

	m := &dto.Metric{}
	if err := counter.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}

	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected counter value 1 for unmatched, got %f", m.GetCounter().GetValue())
	}
}

func TestMetrics_SkipsMetricsEndpoint(t *testing.T) {
	metrics.HTTPRequestsTotal.Reset()

	r := chi.NewRouter()
	r.Use(Metrics)
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// No metrics should be recorded for /metrics itself
	counter, err := metrics.HTTPRequestsTotal.GetMetricWithLabelValues("GET", "/metrics", "200")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	m := &dto.Metric{}
	if err := counter.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if m.GetCounter().GetValue() != 0 {
		t.Errorf("expected 0 for /metrics endpoint, got %f", m.GetCounter().GetValue())
	}
}
