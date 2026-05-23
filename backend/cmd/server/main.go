package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/handler"
	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/logging"
	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/metrics"
	appmiddleware "github.com/ankan-ekansh/Copy-Pasta/backend/internal/middleware"
	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	logging.Init()
	metrics.Register()

	r := chi.NewRouter()
	appmiddleware.Register(r)

	// Connect to database if DATABASE_URL is set
	var s store.Store
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var err error
		s, err = store.NewPostgres(ctx, dbURL)
		if err != nil {
			slog.Warn("failed to connect to database, persistence disabled", "error", err)
		} else {
			s = store.NewInstrumented(s)
			defer s.Close()
			slog.Info("connected to database")
		}
	} else {
		slog.Info("DATABASE_URL not set, persistence disabled")
	}

	h := handler.New(handler.WithStore(s))
	r.Route("/api", func(api chi.Router) {
		// /api/convert has its own stricter rate limit — excluded from RateLimitAPI
		api.With(appmiddleware.RateLimitConvert()).Post("/convert", h.Convert)

		// All other API routes share the general rate limit
		api.Group(func(general chi.Router) {
			general.Use(appmiddleware.RateLimitAPI())
			general.Get("/health", h.Health)
			general.Get("/pastas", h.ListPastas)
			general.Get("/pastas/{id}", h.GetPasta)
			general.Delete("/pastas/{id}", h.DeletePasta)
			general.Patch("/pastas/{id}", h.SetPublic)
		})
	})

	// Expose /metrics endpoint for Prometheus scraping.
	// Requires EXPOSE_METRICS=true (or any truthy value: 1, t, yes) to mount the endpoint.
	// If METRICS_TOKEN is set, requires Authorization: Bearer <token> header.
	// Note: internal instrumentation (middleware/store metrics) still runs regardless;
	// this flag only controls whether the /metrics HTTP endpoint is reachable.
	if exposeMetrics, err := strconv.ParseBool(os.Getenv("EXPOSE_METRICS")); err == nil && exposeMetrics {
		metricsHandler := promhttp.Handler()
		if token := os.Getenv("METRICS_TOKEN"); token != "" {
			metricsHandler = requireBearerToken(token, metricsHandler)
			slog.Info("metrics endpoint enabled with token auth", "path", "/metrics")
		} else {
			slog.Info("metrics endpoint enabled (no auth)", "path", "/metrics")
		}
		r.Handle("/metrics", metricsHandler)
	}

	// Serve static frontend files if the directory exists (production mode)
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "static"
	}
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		slog.Info("serving static files", "dir", staticDir)
		spaHandler := spaFileServer(os.DirFS(staticDir))
		r.NotFound(spaHandler)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	slog.Info("starting server", "addr", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// spaFileServer serves static files and falls back to index.html for SPA routing.
func spaFileServer(root fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(root))
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Try to serve the file directly
		if _, err := fs.Stat(root, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fall back to index.html for SPA client-side routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	}
}

// requireBearerToken wraps a handler to require a valid Authorization: Bearer <token> header.
func requireBearerToken(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != token {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
