package database

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type MomentStatus string

const (
	StatusIntake    MomentStatus = "intake"
	StatusDrafting  MomentStatus = "drafting"
	StatusRendering MomentStatus = "rendering"
	StatusReady     MomentStatus = "ready"
	StatusFailed    MomentStatus = "failed"
)

type JobType string

const (
	JobCutDraft JobType = "cut-draft"
	JobRender   JobType = "render"
)

type JobStatus string

const (
	JobQueued     JobStatus = "queued"
	JobRunning    JobStatus = "running"
	JobSucceeded  JobStatus = "succeeded"
	JobFailed     JobStatus = "failed"
	JobSuperseded JobStatus = "superseded"
)

type Moment struct {
	ID               uuid.UUID
	UserID           string
	Status           MomentStatus
	SongPickID       string
	AssetIDs         []uuid.UUID
	CutID            *uuid.UUID
	RenderStorageKey *string
	ErrorCode        *string
	ErrorMessage     *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Cut struct {
	ID         uuid.UUID
	MomentID   uuid.UUID
	Version    int
	SongPickID string
	Clips      json.RawMessage
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Job struct {
	ID             uuid.UUID
	MomentID       uuid.UUID
	JobType        JobType
	Status         JobStatus
	Attempt        int
	Payload        json.RawMessage
	Result         json.RawMessage
	ErrorCode      *string
	ErrorMessage   *string
	CutVersion     *int
	IdempotencyKey *string
	ClaimedBy      *string
	ClaimedAt      *time.Time
	LeaseUntil     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type IdempotencyRecord struct {
	UserID      string
	Key         string
	RequestHash string
	MomentID    uuid.UUID
	CreatedAt   time.Time
}
