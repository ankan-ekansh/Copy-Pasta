package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/store"
	"github.com/go-chi/chi/v5"

	appmiddleware "github.com/ankan-ekansh/Copy-Pasta/backend/internal/middleware"
)

type pastaResponse struct {
	ID        string `json:"id"`
	ASCIIArt  string `json:"ascii_art"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Mode      string `json:"mode"`
	IsPublic  bool   `json:"is_public"`
	CreatedAt string `json:"created_at"`
}

type listResponse struct {
	Pastas []pastaResponse `json:"pastas"`
}

// GetPasta handles GET /api/pastas/:id — anyone with the link can view.
func (h *Handler) GetPasta(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, pastaResponse{
		ID:        pasta.ID,
		ASCIIArt:  pasta.ASCIIArt,
		Width:     pasta.Width,
		Height:    pasta.Height,
		Mode:      pasta.Mode,
		IsPublic:  pasta.IsPublic,
		CreatedAt: pasta.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// ListPastas handles GET /api/pastas — returns pastas for the current session.
func (h *Handler) ListPastas(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeError(w, http.StatusServiceUnavailable, "persistence not configured")
		return
	}

	sessionID := appmiddleware.GetSessionID(r.Context())
	if sessionID == "" {
		writeError(w, http.StatusUnauthorized, "session required")
		return
	}

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

	pastas, err := h.store.ListBySession(r.Context(), sessionID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	items := make([]pastaResponse, 0, len(pastas))
	for _, p := range pastas {
		items = append(items, pastaResponse{
			ID:        p.ID,
			ASCIIArt:  p.ASCIIArt,
			Width:     p.Width,
			Height:    p.Height,
			Mode:      p.Mode,
			IsPublic:  p.IsPublic,
			CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, listResponse{Pastas: items})
}

// DeletePasta handles DELETE /api/pastas/:id — owner only.
func (h *Handler) DeletePasta(w http.ResponseWriter, r *http.Request) {
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

	// Verify ownership
	pasta, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "pasta not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if pasta.SessionID != sessionID {
		writeError(w, http.StatusForbidden, "not your pasta")
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "pasta not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SetPublic handles PATCH /api/pastas/:id — toggle is_public (owner only).
func (h *Handler) SetPublic(w http.ResponseWriter, r *http.Request) {
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

	// Verify ownership
	pasta, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "pasta not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if pasta.SessionID != sessionID {
		writeError(w, http.StatusForbidden, "not your pasta")
		return
	}

	var body struct {
		IsPublic bool `json:"is_public"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := h.store.SetPublic(r.Context(), id, body.IsPublic); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "pasta not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"is_public": body.IsPublic})
}
