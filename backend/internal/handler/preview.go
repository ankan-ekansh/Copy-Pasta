package handler

import (
	"net/http"

	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/preview"
	"github.com/go-chi/chi/v5"
)

// PreviewImage renders a pasta's ASCII art as a PNG for OG image previews.
func (h *Handler) PreviewImage(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	pasta, err := h.store.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	renderer := h.renderer
	if renderer == nil {
		http.Error(w, "preview not available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if err := renderer.Render(w, pasta.ASCIIArt); err != nil {
		http.Error(w, "failed to render preview", http.StatusInternalServerError)
		return
	}
}

// WithRenderer configures the handler with a preview renderer.
func WithRenderer(r *preview.Renderer) Option {
	return func(h *Handler) {
		h.renderer = r
	}
}
