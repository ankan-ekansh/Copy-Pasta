// Package store provides persistence for pasta conversions.
package store

import (
	"context"
	"time"
)

// Pasta represents a saved ASCII/Braille art conversion.
type Pasta struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	ASCIIArt  string    `json:"ascii_art"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Mode      string    `json:"mode"`
	IsPublic  bool      `json:"is_public"`
	CreatedAt time.Time `json:"created_at"`
}

// GalleryPasta holds public pasta data with social metadata for gallery views.
// Does not embed Pasta to prevent accidental SessionID serialization.
type GalleryPasta struct {
	ID        string    `json:"id"`
	ASCIIArt  string    `json:"ascii_art"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Mode      string    `json:"mode"`
	IsPublic  bool      `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	LikeCount int       `json:"like_count"`
	LikedByMe bool      `json:"liked_by_me"`
}

// Store defines the persistence interface for pastas.
type Store interface {
	// Save persists a new pasta. The caller must set p.ID before calling.
	Save(ctx context.Context, p *Pasta) error

	// Get retrieves a pasta by ID. Returns ErrNotFound if it doesn't exist.
	Get(ctx context.Context, id string) (*Pasta, error)

	// ListBySession returns pastas for a given session, ordered by created_at desc.
	ListBySession(ctx context.Context, sessionID string, limit, offset int) ([]Pasta, error)

	// ListPublic returns public pastas for the gallery, ordered by created_at desc.
	// Includes like counts and whether the given session has liked each pasta.
	ListPublic(ctx context.Context, sessionID string, limit, offset int) ([]GalleryPasta, error)

	// DeleteByOwner removes a pasta only if it belongs to the given session.
	// Returns ErrNotFound if the pasta doesn't exist or isn't owned by the session.
	DeleteByOwner(ctx context.Context, id, sessionID string) error

	// SetPublicByOwner updates the is_public flag only if the pasta belongs to the session.
	// Returns ErrNotFound if the pasta doesn't exist or isn't owned by the session.
	SetPublicByOwner(ctx context.Context, id, sessionID string, isPublic bool) error

	// LikePasta adds a like from the session. Idempotent — no-op if already liked.
	LikePasta(ctx context.Context, pastaID, sessionID string) error

	// UnlikePasta removes a like from the session. Idempotent — no-op if not liked.
	UnlikePasta(ctx context.Context, pastaID, sessionID string) error

	// GetLikeCount returns the like count for a pasta and whether the session liked it.
	GetLikeCount(ctx context.Context, pastaID, sessionID string) (count int, likedByMe bool, err error)

	// Close releases any resources held by the store.
	Close()
}
