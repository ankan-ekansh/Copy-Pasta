package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ankan/copy-pasta/internal/handler"
	appmiddleware "github.com/ankan/copy-pasta/internal/middleware"
	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	appmiddleware.Register(r)

	h := handler.New()
	r.Get("/api/health", h.Health)
	r.Post("/api/convert", h.Convert)

	// Serve static frontend files if the directory exists (production mode)
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "static"
	}
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		log.Printf("serving static files from %s", staticDir)
		spaHandler := spaFileServer(os.DirFS(staticDir))
		r.NotFound(spaHandler)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("starting server on %s", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server failed: %v", err)
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
