package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("not found")

type CreateAssetParams struct {
	ID           uuid.UUID
	UserID       string
	Type         AssetType
	MimeType     string
	ByteSize     int64
	StorageKey   string
	ThumbnailKey *string
	DurationMs   *int
	Width        *int
	Height       *int
}

func (s *Store) CreateAsset(ctx context.Context, p CreateAssetParams) (*Asset, error) {
	const q = `
		INSERT INTO assets (
			id, user_id, type, status, mime_type, byte_size, storage_key,
			thumbnail_key, duration_ms, width, height
		) VALUES (
			$1, $2, $3, 'pending', $4, $5, $6, $7, $8, $9, $10
		)
		RETURNING id, user_id, type, status, mime_type, byte_size, storage_key,
			thumbnail_key, duration_ms, width, height, checksum, created_at, updated_at`
	row := s.pool.QueryRow(ctx, q,
		p.ID, p.UserID, p.Type, p.MimeType, p.ByteSize, p.StorageKey,
		p.ThumbnailKey, p.DurationMs, p.Width, p.Height,
	)
	return scanAsset(row)
}

func (s *Store) GetAsset(ctx context.Context, id uuid.UUID) (*Asset, error) {
	const q = `
		SELECT id, user_id, type, status, mime_type, byte_size, storage_key,
			thumbnail_key, duration_ms, width, height, checksum, created_at, updated_at
		FROM assets WHERE id = $1`
	a, err := scanAsset(s.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (s *Store) ListAssetsByIDs(ctx context.Context, userID string, ids []uuid.UUID) ([]Asset, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	const q = `
		SELECT id, user_id, type, status, mime_type, byte_size, storage_key,
			thumbnail_key, duration_ms, width, height, checksum, created_at, updated_at
		FROM assets
		WHERE user_id = $1 AND id = ANY($2::uuid[])`
	rows, err := s.pool.Query(ctx, q, userID, uuidSlice(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Asset
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (s *Store) ListAssetsByIDsAny(ctx context.Context, ids []uuid.UUID) ([]Asset, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	const q = `
		SELECT id, user_id, type, status, mime_type, byte_size, storage_key,
			thumbnail_key, duration_ms, width, height, checksum, created_at, updated_at
		FROM assets
		WHERE id = ANY($1::uuid[])`
	rows, err := s.pool.Query(ctx, q, uuidSlice(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Asset
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func uuidSlice(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

type CompleteAssetParams struct {
	ID         uuid.UUID
	ByteSize   int64
	Checksum   *string
	DurationMs *int
	Width      *int
	Height     *int
	// ClearThumbnail drops thumbnail_key when poster was never uploaded.
	ClearThumbnail bool
}

func (s *Store) MarkAssetReady(ctx context.Context, p CompleteAssetParams) (*Asset, error) {
	const q = `
		UPDATE assets SET
			status = 'ready',
			byte_size = $2,
			checksum = COALESCE($3, checksum),
			duration_ms = COALESCE($4, duration_ms),
			width = COALESCE($5, width),
			height = COALESCE($6, height),
			thumbnail_key = CASE WHEN $7 THEN NULL ELSE thumbnail_key END,
			updated_at = now()
		WHERE id = $1 AND status = 'pending'
		RETURNING id, user_id, type, status, mime_type, byte_size, storage_key,
			thumbnail_key, duration_ms, width, height, checksum, created_at, updated_at`
	a, err := scanAsset(s.pool.QueryRow(ctx, q,
		p.ID, p.ByteSize, p.Checksum, p.DurationMs, p.Width, p.Height, p.ClearThumbnail,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (s *Store) MarkAssetFailed(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE assets SET status = 'failed', updated_at = now()
		WHERE id = $1 AND status = 'pending'`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteAsset(ctx context.Context, id uuid.UUID, userID string) (*Asset, error) {
	const q = `
		DELETE FROM assets
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, type, status, mime_type, byte_size, storage_key,
			thumbnail_key, duration_ms, width, height, checksum, created_at, updated_at`
	a, err := scanAsset(s.pool.QueryRow(ctx, q, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (s *Store) CountRecentCreates(ctx context.Context, userID string, since time.Time) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM assets
		WHERE user_id = $1 AND created_at >= $2`, userID, since,
	).Scan(&n)
	return n, err
}

type scannable interface {
	Scan(dest ...any) error
}

func scanAsset(row scannable) (*Asset, error) {
	var a Asset
	var typ, status string
	err := row.Scan(
		&a.ID, &a.UserID, &typ, &status, &a.MimeType, &a.ByteSize, &a.StorageKey,
		&a.ThumbnailKey, &a.DurationMs, &a.Width, &a.Height, &a.Checksum,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	a.Type = AssetType(typ)
	a.Status = AssetStatus(status)
	return &a, nil
}

func (s *Store) Ping(ctx context.Context) error {
	if err := s.pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}
	return nil
}
