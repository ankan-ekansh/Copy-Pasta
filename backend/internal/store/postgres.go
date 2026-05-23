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
	statements := []string{
		`CREATE TABLE IF NOT EXISTS pastas (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			ascii_art TEXT NOT NULL,
			width INT NOT NULL,
			height INT NOT NULL,
			mode TEXT NOT NULL,
			is_public BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pastas_session ON pastas(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pastas_public ON pastas(is_public, created_at)`,
		`CREATE TABLE IF NOT EXISTS likes (
			pasta_id TEXT NOT NULL REFERENCES pastas(id) ON DELETE CASCADE,
			session_id TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (pasta_id, session_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_likes_session ON likes(session_id)`,
	}
	for _, stmt := range statements {
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
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

func (s *PostgresStore) DeleteByOwner(ctx context.Context, id, sessionID string) error {
	result, err := s.pool.Exec(ctx, "DELETE FROM pastas WHERE id = $1 AND session_id = $2", id, sessionID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) SetPublicByOwner(ctx context.Context, id, sessionID string, isPublic bool) error {
	result, err := s.pool.Exec(ctx, "UPDATE pastas SET is_public = $1 WHERE id = $2 AND session_id = $3", isPublic, id, sessionID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ListPublic(ctx context.Context, sessionID string, limit, offset int) ([]GalleryPasta, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// CTE paginates pastas first, then LEFT JOIN likes only for the page.
	query := `
		WITH page AS (
			SELECT id, ascii_art, width, height, mode, created_at
			FROM pastas
			WHERE is_public = TRUE
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		)
		SELECT page.id, page.ascii_art, page.width, page.height, page.mode, page.created_at,
			COUNT(l.session_id) AS like_count,
			COALESCE(BOOL_OR(l.session_id = $3), FALSE) AS liked_by_me
		FROM page
		LEFT JOIN likes l ON l.pasta_id = page.id
		GROUP BY page.id, page.ascii_art, page.width, page.height, page.mode, page.created_at
		ORDER BY page.created_at DESC
	`
	rows, err := s.pool.Query(ctx, query, limit, offset, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pastas []GalleryPasta
	for rows.Next() {
		var gp GalleryPasta
		if err := rows.Scan(&gp.ID, &gp.ASCIIArt, &gp.Width, &gp.Height,
			&gp.Mode, &gp.CreatedAt, &gp.LikeCount, &gp.LikedByMe); err != nil {
			return nil, err
		}
		gp.IsPublic = true
		pastas = append(pastas, gp)
	}
	return pastas, rows.Err()
}

func (s *PostgresStore) LikePasta(ctx context.Context, pastaID, sessionID string) error {
	result, err := s.pool.Exec(ctx,
		`INSERT INTO likes (pasta_id, session_id)
		 SELECT $1, $2 FROM pastas WHERE id = $1 AND is_public = TRUE
		 ON CONFLICT DO NOTHING`,
		pastaID, sessionID)
	if err != nil {
		return err
	}
	// No row inserted means pasta doesn't exist or isn't public
	if result.RowsAffected() == 0 {
		// Check if it's already liked (ON CONFLICT DO NOTHING)
		var exists bool
		err = s.pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM likes WHERE pasta_id = $1 AND session_id = $2)",
			pastaID, sessionID).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			return nil // idempotent: already liked
		}
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) UnlikePasta(ctx context.Context, pastaID, sessionID string) error {
	_, err := s.pool.Exec(ctx,
		"DELETE FROM likes WHERE pasta_id = $1 AND session_id = $2",
		pastaID, sessionID)
	return err
}

func (s *PostgresStore) GetLikeCount(ctx context.Context, pastaID, sessionID string) (int, bool, error) {
	var count int
	var likedByMe bool
	err := s.pool.QueryRow(ctx, `
		SELECT 
			(SELECT COUNT(*) FROM likes WHERE pasta_id = $1),
			EXISTS(SELECT 1 FROM likes WHERE pasta_id = $1 AND session_id = $2)
	`, pastaID, sessionID).Scan(&count, &likedByMe)
	if err != nil {
		return 0, false, err
	}
	return count, likedByMe, nil
}

func (s *PostgresStore) IsPublicPasta(ctx context.Context, id string) error {
	var exists bool
	err := s.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM pastas WHERE id = $1 AND is_public = TRUE)",
		id).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}

// Ping checks database connectivity.
func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
