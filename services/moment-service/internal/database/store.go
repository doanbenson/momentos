package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrIdempotencyConflict = errors.New("idempotency key reuse with different body")
	ErrActiveJobExists     = errors.New("active job already exists")
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// --- Moments ---

func (s *Store) InsertMoment(ctx context.Context, m *Moment) error {
	const q = `
INSERT INTO moments (id, user_id, status, song_pick_id, asset_ids, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, now(), now())
RETURNING created_at, updated_at`
	return s.pool.QueryRow(ctx, q, m.ID, m.UserID, m.Status, m.SongPickID, m.AssetIDs).
		Scan(&m.CreatedAt, &m.UpdatedAt)
}

func (s *Store) GetMoment(ctx context.Context, id uuid.UUID) (*Moment, error) {
	const q = `
SELECT id, user_id, status, song_pick_id, asset_ids, cut_id, render_storage_key,
       error_code, error_message, created_at, updated_at
FROM moments WHERE id = $1`
	return scanMoment(s.pool.QueryRow(ctx, q, id))
}

func (s *Store) GetMomentForUser(ctx context.Context, id uuid.UUID, userID string) (*Moment, error) {
	const q = `
SELECT id, user_id, status, song_pick_id, asset_ids, cut_id, render_storage_key,
       error_code, error_message, created_at, updated_at
FROM moments WHERE id = $1 AND user_id = $2`
	m, err := scanMoment(s.pool.QueryRow(ctx, q, id, userID))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

func (s *Store) UpdateMomentAssetIDs(ctx context.Context, id uuid.UUID, assetIDs []uuid.UUID) error {
	const q = `UPDATE moments SET asset_ids = $2, updated_at = now() WHERE id = $1`
	ct, err := s.pool.Exec(ctx, q, id, assetIDs)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SetMomentStatus(ctx context.Context, id uuid.UUID, status MomentStatus, errCode, errMsg *string) error {
	const q = `
UPDATE moments
SET status = $2, error_code = $3, error_message = $4, updated_at = now()
WHERE id = $1`
	ct, err := s.pool.Exec(ctx, q, id, status, errCode, errMsg)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SetMomentReady(ctx context.Context, id uuid.UUID, renderKey string) error {
	const q = `
UPDATE moments
SET status = 'ready', render_storage_key = $2, error_code = NULL, error_message = NULL, updated_at = now()
WHERE id = $1`
	ct, err := s.pool.Exec(ctx, q, id, renderKey)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) AttachCut(ctx context.Context, momentID, cutID uuid.UUID) error {
	const q = `
UPDATE moments
SET cut_id = $2, status = 'rendering', error_code = NULL, error_message = NULL, updated_at = now()
WHERE id = $1`
	ct, err := s.pool.Exec(ctx, q, momentID, cutID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanMoment(row scannable) (*Moment, error) {
	var m Moment
	err := row.Scan(
		&m.ID, &m.UserID, &m.Status, &m.SongPickID, &m.AssetIDs, &m.CutID,
		&m.RenderStorageKey, &m.ErrorCode, &m.ErrorMessage, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// --- Cuts ---

func (s *Store) GetCutByMoment(ctx context.Context, momentID uuid.UUID) (*Cut, error) {
	const q = `
SELECT id, moment_id, version, song_pick_id, clips, created_at, updated_at
FROM cuts WHERE moment_id = $1`
	return scanCut(s.pool.QueryRow(ctx, q, momentID))
}

func (s *Store) UpsertCut(ctx context.Context, momentID uuid.UUID, songPickID string, clips json.RawMessage) (*Cut, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var existingID uuid.UUID
	var version int
	err = tx.QueryRow(ctx, `SELECT id, version FROM cuts WHERE moment_id = $1 FOR UPDATE`, momentID).
		Scan(&existingID, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		c := &Cut{
			ID:         uuid.New(),
			MomentID:   momentID,
			Version:    1,
			SongPickID: songPickID,
			Clips:      clips,
		}
		const ins = `
INSERT INTO cuts (id, moment_id, version, song_pick_id, clips, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, now(), now())
RETURNING created_at, updated_at`
		if err := tx.QueryRow(ctx, ins, c.ID, c.MomentID, c.Version, c.SongPickID, c.Clips).
			Scan(&c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return c, nil
	}
	if err != nil {
		return nil, err
	}

	c := &Cut{
		ID:         existingID,
		MomentID:   momentID,
		Version:    version + 1,
		SongPickID: songPickID,
		Clips:      clips,
	}
	const upd = `
UPDATE cuts
SET version = $2, song_pick_id = $3, clips = $4, updated_at = now()
WHERE id = $1
RETURNING created_at, updated_at`
	if err := tx.QueryRow(ctx, upd, c.ID, c.Version, c.SongPickID, c.Clips).
		Scan(&c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

func scanCut(row scannable) (*Cut, error) {
	var c Cut
	err := row.Scan(&c.ID, &c.MomentID, &c.Version, &c.SongPickID, &c.Clips, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if c.Clips == nil {
		c.Clips = json.RawMessage("[]")
	}
	return &c, nil
}

// --- Jobs ---

func (s *Store) SupersedeActiveJobs(ctx context.Context, momentID uuid.UUID, jobType JobType) error {
	const q = `
UPDATE jobs
SET status = 'superseded', updated_at = now()
WHERE moment_id = $1 AND job_type = $2 AND status IN ('queued', 'running')`
	_, err := s.pool.Exec(ctx, q, momentID, jobType)
	return err
}

func (s *Store) InsertJob(ctx context.Context, j *Job) error {
	const q = `
INSERT INTO jobs (
  id, moment_id, job_type, status, attempt, payload, cut_version, idempotency_key, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), now())
RETURNING created_at, updated_at`
	err := s.pool.QueryRow(ctx, q,
		j.ID, j.MomentID, j.JobType, j.Status, j.Attempt, j.Payload, j.CutVersion, j.IdempotencyKey,
	).Scan(&j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrActiveJobExists
		}
		return err
	}
	return nil
}

func (s *Store) ClaimJob(ctx context.Context, jobType JobType, workerID string, lease time.Duration) (*Job, error) {
	const q = `
UPDATE jobs
SET status = 'running',
    claimed_by = $2,
    claimed_at = now(),
    lease_until = now() + $3::interval,
    attempt = attempt + 1,
    updated_at = now()
WHERE id = (
  SELECT id FROM jobs
  WHERE job_type = $1
    AND (
      status = 'queued'
      OR (status = 'running' AND lease_until IS NOT NULL AND lease_until < now())
    )
  ORDER BY created_at
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
RETURNING id, moment_id, job_type, status, attempt, payload, result, error_code, error_message,
          cut_version, idempotency_key, claimed_by, claimed_at, lease_until, created_at, updated_at`
	interval := fmt.Sprintf("%d seconds", int(lease.Seconds()))
	j, err := scanJob(s.pool.QueryRow(ctx, q, jobType, workerID, interval))
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return j, err
}

func (s *Store) HeartbeatJob(ctx context.Context, jobID uuid.UUID, workerID string, lease time.Duration) error {
	const q = `
UPDATE jobs
SET lease_until = now() + $3::interval, updated_at = now()
WHERE id = $1 AND claimed_by = $2 AND status = 'running'`
	interval := fmt.Sprintf("%d seconds", int(lease.Seconds()))
	ct, err := s.pool.Exec(ctx, q, jobID, workerID, interval)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) GetJob(ctx context.Context, id uuid.UUID) (*Job, error) {
	const q = `
SELECT id, moment_id, job_type, status, attempt, payload, result, error_code, error_message,
       cut_version, idempotency_key, claimed_by, claimed_at, lease_until, created_at, updated_at
FROM jobs WHERE id = $1`
	return scanJob(s.pool.QueryRow(ctx, q, id))
}

func (s *Store) CompleteJobSuccess(ctx context.Context, jobID uuid.UUID, result json.RawMessage) error {
	const q = `
UPDATE jobs
SET status = 'succeeded', result = $2, error_code = NULL, error_message = NULL, updated_at = now()
WHERE id = $1 AND status = 'running'`
	ct, err := s.pool.Exec(ctx, q, jobID, result)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CompleteJobFailure(ctx context.Context, jobID uuid.UUID, code, message string) error {
	const q = `
UPDATE jobs
SET status = 'failed', error_code = $2, error_message = $3, updated_at = now()
WHERE id = $1 AND status IN ('running', 'queued')`
	ct, err := s.pool.Exec(ctx, q, jobID, code, message)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanJob(row scannable) (*Job, error) {
	var j Job
	err := row.Scan(
		&j.ID, &j.MomentID, &j.JobType, &j.Status, &j.Attempt, &j.Payload, &j.Result,
		&j.ErrorCode, &j.ErrorMessage, &j.CutVersion, &j.IdempotencyKey,
		&j.ClaimedBy, &j.ClaimedAt, &j.LeaseUntil, &j.CreatedAt, &j.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// --- Idempotency ---

func (s *Store) GetIdempotency(ctx context.Context, userID, key string) (*IdempotencyRecord, error) {
	const q = `
SELECT user_id, key, request_hash, moment_id, created_at
FROM idempotency_keys WHERE user_id = $1 AND key = $2`
	var r IdempotencyRecord
	err := s.pool.QueryRow(ctx, q, userID, key).
		Scan(&r.UserID, &r.Key, &r.RequestHash, &r.MomentID, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) PutIdempotency(ctx context.Context, userID, key, requestHash string, momentID uuid.UUID) error {
	const q = `
INSERT INTO idempotency_keys (user_id, key, request_hash, moment_id, created_at)
VALUES ($1, $2, $3, $4, now())`
	_, err := s.pool.Exec(ctx, q, userID, key, requestHash, momentID)
	return err
}

// CreateMomentTx inserts moment + optional idempotency + initial cut-draft job atomically.
func (s *Store) CreateMomentWithJob(ctx context.Context, m *Moment, job *Job, idem *IdempotencyRecord) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	const insM = `
INSERT INTO moments (id, user_id, status, song_pick_id, asset_ids, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, now(), now())
RETURNING created_at, updated_at`
	if err := tx.QueryRow(ctx, insM, m.ID, m.UserID, m.Status, m.SongPickID, m.AssetIDs).
		Scan(&m.CreatedAt, &m.UpdatedAt); err != nil {
		return err
	}

	const insJ = `
INSERT INTO jobs (
  id, moment_id, job_type, status, attempt, payload, cut_version, idempotency_key, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), now())
RETURNING created_at, updated_at`
	if err := tx.QueryRow(ctx, insJ,
		job.ID, job.MomentID, job.JobType, job.Status, job.Attempt, job.Payload, job.CutVersion, job.IdempotencyKey,
	).Scan(&job.CreatedAt, &job.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrActiveJobExists
		}
		return err
	}

	if idem != nil {
		const insI = `
INSERT INTO idempotency_keys (user_id, key, request_hash, moment_id, created_at)
VALUES ($1, $2, $3, $4, now())`
		if _, err := tx.Exec(ctx, insI, idem.UserID, idem.Key, idem.RequestHash, idem.MomentID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
