package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
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
	r.Get("/api/health", h.Health)
	r.Post("/api/convert", h.Convert)
	r.Get("/api/pastas", h.ListPastas)
	r.Get("/api/pastas/{id}", h.GetPasta)
	r.Delete("/api/pastas/{id}", h.DeletePasta)
	r.Patch("/api/pastas/{id}", h.SetPublic)

	// Expose /metrics endpoint for Prometheus scraping.
	// Requires EXPOSE_METRICS=true to mount the endpoint (opt-in).
	// Note: internal instrumentation (middleware/store metrics) still runs regardless;
	// this flag only controls whether the /metrics HTTP endpoint is reachable.
	if os.Getenv("EXPOSE_METRICS") == "true" {
		r.Handle("/metrics", promhttp.Handler())
		slog.Info("metrics endpoint enabled", "path", "/metrics")
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
