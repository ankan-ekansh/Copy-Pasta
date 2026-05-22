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

// Store defines the persistence interface for pastas.
type Store interface {
	// Save persists a new pasta and returns its ID.
	Save(ctx context.Context, p *Pasta) error

	// Get retrieves a pasta by ID. Returns ErrNotFound if it doesn't exist.
	Get(ctx context.Context, id string) (*Pasta, error)

	// ListBySession returns pastas for a given session, ordered by created_at desc.
	ListBySession(ctx context.Context, sessionID string, limit, offset int) ([]Pasta, error)

	// Delete removes a pasta. Only the owner (by session) should call this.
	Delete(ctx context.Context, id string) error

	// SetPublic updates the is_public flag.
	SetPublic(ctx context.Context, id string, isPublic bool) error

	// Close releases any resources held by the store.
	Close()
}
