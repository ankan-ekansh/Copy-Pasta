package handler

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/preview"
	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/store"
	"github.com/go-chi/chi/v5"
)

// PreviewImage renders a pasta's ASCII art as a PNG for OG image previews.
func (h *Handler) PreviewImage(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeError(w, http.StatusServiceUnavailable, "persistence not configured")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "pasta ID is required")
		return
	}

	pasta, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "pasta not found")
			return
		}
		slog.Error("preview: failed to fetch pasta", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	renderer := h.renderer
	if renderer == nil {
		writeError(w, http.StatusServiceUnavailable, "preview not available")
		return
	}

	// Render into buffer first to avoid partial/corrupted responses on error
	var buf bytes.Buffer
	if err := renderer.Render(&buf, pasta.ASCIIArt); err != nil {
		slog.Error("preview: render failed", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to render preview")
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(buf.Bytes())
}

// WithRenderer configures the handler with a preview renderer.
func WithRenderer(r *preview.Renderer) Option {
	return func(h *Handler) {
		h.renderer = r
	}
}
