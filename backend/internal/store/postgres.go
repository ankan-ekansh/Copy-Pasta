package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgres creates a new PostgresStore, connects to the database, and runs migrations.
func NewPostgres(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	s := &PostgresStore{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return s, nil
}

func (s *PostgresStore) migrate(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS pastas (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			ascii_art TEXT NOT NULL,
			width INT NOT NULL,
			height INT NOT NULL,
			mode TEXT NOT NULL,
			is_public BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_pastas_session ON pastas(session_id);
		CREATE INDEX IF NOT EXISTS idx_pastas_public ON pastas(is_public, created_at);
	`
	_, err := s.pool.Exec(ctx, query)
	return err
}

func (s *PostgresStore) Save(ctx context.Context, p *Pasta) error {
	query := `
		INSERT INTO pastas (id, session_id, ascii_art, width, height, mode, is_public)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at
	`
	return s.pool.QueryRow(ctx, query, p.ID, p.SessionID, p.ASCIIArt, p.Width, p.Height, p.Mode, p.IsPublic).Scan(&p.CreatedAt)
}

func (s *PostgresStore) Get(ctx context.Context, id string) (*Pasta, error) {
	query := `
		SELECT id, session_id, ascii_art, width, height, mode, is_public, created_at
		FROM pastas WHERE id = $1
	`
	row := s.pool.QueryRow(ctx, query, id)

	var p Pasta
	err := row.Scan(&p.ID, &p.SessionID, &p.ASCIIArt, &p.Width, &p.Height, &p.Mode, &p.IsPublic, &p.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (s *PostgresStore) ListBySession(ctx context.Context, sessionID string, limit, offset int) ([]Pasta, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT id, session_id, ascii_art, width, height, mode, is_public, created_at
		FROM pastas
		WHERE session_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := s.pool.Query(ctx, query, sessionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pastas []Pasta
	for rows.Next() {
		var p Pasta
		if err := rows.Scan(&p.ID, &p.SessionID, &p.ASCIIArt, &p.Width, &p.Height, &p.Mode, &p.IsPublic, &p.CreatedAt); err != nil {
			return nil, err
		}
		pastas = append(pastas, p)
	}
	return pastas, rows.Err()
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	result, err := s.pool.Exec(ctx, "DELETE FROM pastas WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) SetPublic(ctx context.Context, id string, isPublic bool) error {
	result, err := s.pool.Exec(ctx, "UPDATE pastas SET is_public = $1 WHERE id = $2", isPublic, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}
