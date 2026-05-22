package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/handler"
	appmiddleware "github.com/ankan-ekansh/Copy-Pasta/backend/internal/middleware"
	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/store"
	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	appmiddleware.Register(r)

	// Connect to database if DATABASE_URL is set
	var s store.Store
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		var err error
		s, err = store.NewPostgres(context.Background(), dbURL)
		if err != nil {
			log.Fatalf("failed to connect to database: %v", err)
		}
		defer s.Close()
		log.Println("connected to database")
	} else {
		log.Println("DATABASE_URL not set, persistence disabled")
	}

	h := handler.New(handler.WithStore(s))
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
