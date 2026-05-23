package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/store"
	"github.com/go-chi/chi/v5"

	appmiddleware "github.com/ankan-ekansh/Copy-Pasta/backend/internal/middleware"
)

type galleryPastaResponse struct {
	ID        string `json:"id"`
	ASCIIArt  string `json:"ascii_art"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Mode      string `json:"mode"`
	LikeCount int    `json:"like_count"`
	LikedByMe bool   `json:"liked_by_me"`
	CreatedAt string `json:"created_at"`
}

type galleryResponse struct {
	Pastas []galleryPastaResponse `json:"pastas"`
}

// ListGallery handles GET /api/gallery — returns public pastas with like counts.
func (h *Handler) ListGallery(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeError(w, http.StatusServiceUnavailable, "persistence not configured")
		return
	}

	sessionID := appmiddleware.GetSessionID(r.Context())

	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	pastas, err := h.store.ListPublic(r.Context(), sessionID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	items := make([]galleryPastaResponse, 0, len(pastas))
	for _, p := range pastas {
		items = append(items, galleryPastaResponse{
			ID:        p.ID,
			ASCIIArt:  p.ASCIIArt,
			Width:     p.Width,
			Height:    p.Height,
			Mode:      p.Mode,
			LikeCount: p.LikeCount,
			LikedByMe: p.LikedByMe,
			CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, galleryResponse{Pastas: items})
}

// LikePasta handles POST /api/pastas/:id/like — adds a like.
func (h *Handler) LikePasta(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeError(w, http.StatusServiceUnavailable, "persistence not configured")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "pasta ID is required")
		return
	}

	sessionID := appmiddleware.GetSessionID(r.Context())
	if sessionID == "" {
		writeError(w, http.StatusUnauthorized, "session required")
		return
	}

	if err := h.store.LikePasta(r.Context(), id, sessionID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "pasta not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	count, _, err := h.store.GetLikeCount(r.Context(), id, sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"like_count":  count,
		"liked_by_me": true,
	})
}

// UnlikePasta handles DELETE /api/pastas/:id/like — removes a like.
func (h *Handler) UnlikePasta(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeError(w, http.StatusServiceUnavailable, "persistence not configured")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "pasta ID is required")
		return
	}

	sessionID := appmiddleware.GetSessionID(r.Context())
	if sessionID == "" {
		writeError(w, http.StatusUnauthorized, "session required")
		return
	}

	// Verify pasta exists and is public before unlike
	pasta, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "pasta not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if !pasta.IsPublic {
		writeError(w, http.StatusNotFound, "pasta not found")
		return
	}

	if err := h.store.UnlikePasta(r.Context(), id, sessionID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	count, _, err := h.store.GetLikeCount(r.Context(), id, sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"like_count":  count,
		"liked_by_me": false,
	})
}
