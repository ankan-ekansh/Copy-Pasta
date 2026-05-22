package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func Register(r chi.Router) {
	r.Use(CORS())
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(Session)
}

func CORS() func(http.Handler) http.Handler {
	// Default: localhost for dev + production URL.
	// In production, frontend is served from same origin so CORS rarely applies.
	// Override with CORS_ORIGINS env var for custom deployments.
	origins := []string{
		"http://localhost:3000",
		"http://localhost:8080",
		"https://copy-pasta.happyflower-831a5c58.eastus.azurecontainerapps.io",
	}
	allowCreds := true
	if env := os.Getenv("CORS_ORIGINS"); env != "" {
		parts := strings.Split(env, ",")
		origins = make([]string, 0, len(parts))
		for _, o := range parts {
			o = strings.TrimSpace(o)
			if o == "" {
				continue
			}
			if o == "*" {
				allowCreds = false
			}
			origins = append(origins, o)
		}
	}
	return cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: allowCreds,
		MaxAge:           300,
	})
}
