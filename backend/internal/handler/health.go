package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/store"
)

var startTime = time.Now()

// version is set via -ldflags "-X github.com/ankan-ekansh/Copy-Pasta/backend/internal/handler.version=..."
// Falls back to APP_VERSION env var, then "dev".
var version = "dev"

func init() {
	if version == "dev" {
		if v := os.Getenv("APP_VERSION"); v != "" {
			version = v
		}
	}
}

type richHealthResponse struct {
	Status        string                  `json:"status"`
	Version       string                  `json:"version"`
	UptimeSeconds int64                   `json:"uptime_seconds"`
	Checks        map[string]*checkResult `json:"checks"`
}

type checkResult struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	resp := richHealthResponse{
		Status:        "healthy",
		Version:       version,
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
		Checks:        make(map[string]*checkResult),
	}

	// Database health check
	if h.store != nil {
		dbCheck := checkDB(r.Context(), h.store)
		resp.Checks["database"] = dbCheck
		if dbCheck.Status == "down" {
			resp.Status = "degraded"
		}
	} else {
		resp.Checks["database"] = &checkResult{Status: "disabled"}
	}

	status := http.StatusOK
	if resp.Status != "healthy" {
		status = http.StatusServiceUnavailable
	}

	writeJSON(w, status, resp)
}

func checkDB(ctx context.Context, s interface{}) *checkResult {
	type pinger interface {
		Ping(ctx context.Context) error
	}

	p, ok := s.(pinger)
	if !ok {
		return &checkResult{Status: "unknown"}
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	start := time.Now()
	err := p.Ping(ctx)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		if errors.Is(err, store.ErrPingNotSupported) {
			return &checkResult{Status: "unknown"}
		}
		slog.Error("health check: database ping failed", "error", err, "latency_ms", latency)
		return &checkResult{Status: "down", LatencyMs: latency}
	}
	return &checkResult{Status: "up", LatencyMs: latency}
}
