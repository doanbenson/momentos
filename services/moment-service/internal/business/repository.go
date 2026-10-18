package business

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/doanbenson/momentos/services/moment-service/internal/database"
)

// Store is the moment-service persistence seam. *database.Store implements it.
type Store interface {
	GetIdempotency(ctx context.Context, userID, key string) (*database.IdempotencyRecord, error)
	GetMoment(ctx context.Context, id uuid.UUID) (*database.Moment, error)
	CreateMomentWithJob(ctx context.Context, m *database.Moment, job *database.Job, idem *database.IdempotencyRecord) error
	GetMomentForUser(ctx context.Context, id uuid.UUID, userID string) (*database.Moment, error)
	ListMomentsForUser(ctx context.Context, userID string) ([]database.Moment, error)
	UpdateMomentAssetIDs(ctx context.Context, id uuid.UUID, assetIDs []uuid.UUID) error
	SetMomentStatus(ctx context.Context, id uuid.UUID, status database.MomentStatus, errCode, errMsg *string) error
	GetCutByMoment(ctx context.Context, momentID uuid.UUID) (*database.Cut, error)
	UpsertCut(ctx context.Context, momentID uuid.UUID, songPickID string, clips, captions, markers json.RawMessage) (*database.Cut, error)
	SupersedeActiveJobs(ctx context.Context, momentID uuid.UUID, jobType database.JobType) error
	InsertJob(ctx context.Context, j *database.Job) error
	ClaimJob(ctx context.Context, jobType database.JobType, workerID string, lease time.Duration) (*database.Job, error)
	HeartbeatJob(ctx context.Context, jobID uuid.UUID, workerID string, lease time.Duration) error
	GetJob(ctx context.Context, id uuid.UUID) (*database.Job, error)
	CompleteJobSuccess(ctx context.Context, jobID uuid.UUID, result json.RawMessage) error
	CompleteJobFailure(ctx context.Context, jobID uuid.UUID, code, message string) error
	AttachCut(ctx context.Context, momentID, cutID uuid.UUID) error
	SetMomentReady(ctx context.Context, id uuid.UUID, renderKey string) error
}
