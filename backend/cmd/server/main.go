package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"html"
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
			general.Get("/gallery", h.ListGallery)
			general.Get("/pastas", h.ListPastas)
			general.Get("/pastas/{id}", h.GetPasta)
			general.Delete("/pastas/{id}", h.DeletePasta)
			general.Patch("/pastas/{id}", h.SetPublic)
			general.Post("/pastas/{id}/like", h.LikePasta)
			general.Delete("/pastas/{id}/like", h.UnlikePasta)
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
		root := os.DirFS(staticDir)
		spaHandler := spaFileServer(root)

		// Serve /pasta/:id with OG meta tags injected (with and without trailing slash)
		ogHandler := ogMetaHandler(root, s)
		r.Get("/pasta/{id}", ogHandler)
		r.Get("/pasta/{id}/", ogHandler)

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

// ogMetaHandler serves index.html with OG meta tags injected for /pasta/:id routes.
// Caches the base HTML at startup. Injects dynamic tags at the <!-- og:dynamic --> placeholder,
// or falls back to plain index.html if the pasta can't be loaded.
func ogMetaHandler(root fs.FS, s store.Store) http.HandlerFunc {
	indexBytes, err := fs.ReadFile(root, "index.html")
	if err != nil {
		slog.Error("og meta: failed to read index.html at startup", "error", err)
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		}
	}
	baseHTML := string(indexBytes)
	const placeholder = "<!-- og:dynamic -->"

	return func(w http.ResponseWriter, r *http.Request) {
		result := baseHTML
		id := chi.URLParam(r, "id")

		if id != "" && s != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()

			pasta, err := s.Get(ctx, id)
			if err == nil {
				ogTags := buildOGMetaTags(pasta, r)
				// Replace placeholder with dynamic tags (scrapers use first occurrence)
				result = strings.Replace(result, placeholder, ogTags, 1)
			} else if !errors.Is(err, store.ErrNotFound) {
				slog.Warn("og meta: failed to fetch pasta", "id", id, "error", err)
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(result))
	}
}

// buildOGMetaTags generates Open Graph and Twitter Card meta tags for a pasta.
func buildOGMetaTags(pasta *store.Pasta, r *http.Request) string {
	scheme := forwardedScheme(r)
	title := "ASCII Art | Copy-Pasta"
	description := fmt.Sprintf("%d×%d %s art — turn memes into text art!", pasta.Width, pasta.Height, pasta.Mode)
	canonicalURL := fmt.Sprintf("%s://%s/pasta/%s", scheme, r.Host, pasta.ID)

	return fmt.Sprintf(`<meta property="og:title" content="%s" />
    <meta property="og:description" content="%s" />
    <meta property="og:url" content="%s" />
    <meta property="og:type" content="website" />
    <meta property="og:site_name" content="Copy-Pasta" />
    <meta name="twitter:card" content="summary" />
    <meta name="twitter:title" content="%s" />
    <meta name="twitter:description" content="%s" />
    `, html.EscapeString(title), html.EscapeString(description), html.EscapeString(canonicalURL), html.EscapeString(title), html.EscapeString(description))
}

// forwardedScheme determines the request scheme from X-Forwarded-Proto (first token),
// falling back to TLS detection.
func forwardedScheme(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		// Take first token only (handles comma-separated lists from chained proxies)
		if idx := strings.IndexAny(proto, ", "); idx > 0 {
			proto = proto[:idx]
		}
		proto = strings.ToLower(strings.TrimSpace(proto))
		if proto == "http" || proto == "https" {
			return proto
		}
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

// requireBearerToken wraps a handler to require a valid Authorization: Bearer <token> header.
// The scheme comparison is case-insensitive per RFC 7235.
// Compares SHA-256 digests to ensure constant-time regardless of input length.
func requireBearerToken(token string, next http.Handler) http.Handler {
	expectedHash := sha256.Sum256([]byte(token))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		// RFC 7235: auth scheme is case-insensitive
		if len(auth) < 7 || !strings.EqualFold(auth[:7], "bearer ") {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		providedHash := sha256.Sum256([]byte(auth[7:]))
		if subtle.ConstantTimeCompare(expectedHash[:], providedHash[:]) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
