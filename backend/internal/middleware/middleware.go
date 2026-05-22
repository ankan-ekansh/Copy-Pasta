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
	origins := []string{"http://localhost:3000", "http://localhost:8080"}
	allowCreds := true
	if env := os.Getenv("CORS_ORIGINS"); env != "" {
		origins = strings.Split(env, ",")
		// Wildcard is incompatible with credentials — disable creds if * is used
		for _, o := range origins {
			if strings.TrimSpace(o) == "*" {
				allowCreds = false
				break
			}
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
