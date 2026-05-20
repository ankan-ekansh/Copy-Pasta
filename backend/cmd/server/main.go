package main

import (
	"log"
	"net/http"
	"os"

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
