package business

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/doanbenson/momentos/services/moment-service/internal/database"
)

type CutDraftPayload struct {
	JobType            string       `json:"jobType"`
	JobID              uuid.UUID    `json:"jobId"`
	MomentID           uuid.UUID    `json:"momentId"`
	SongPickID         string       `json:"songPickId"`
	EditAudioURL       string       `json:"editAudioUrl,omitempty"`
	Mode               string       `json:"mode"`
	Assets             []JobAsset   `json:"assets"`
	ReflectionAssetID  uuid.UUID    `json:"reflectionAssetId,omitempty"`
	ReflectionQuestion string       `json:"reflectionQuestion,omitempty"`
	PriorCut           *CutDTO      `json:"priorCut"`
	EditorPatch        *EditorPatch `json:"editorPatch"`
}

type JobAsset struct {
	ID         uuid.UUID `json:"id"`
	Type       string    `json:"type"`
	Order      int       `json:"order"`
	DurationMs int       `json:"durationMs"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	StorageURL string    `json:"storageUrl"`
}

type RenderPayload struct {
	JobType            string     `json:"jobType"`
	JobID              uuid.UUID  `json:"jobId"`
	MomentID           uuid.UUID  `json:"momentId"`
	CutVersion         int        `json:"cutVersion"`
	Cut                CutDTO     `json:"cut"`
	SongPickID         string     `json:"songPickId"`
	EditAudioURL       string     `json:"editAudioUrl,omitempty"`
	ReflectionQuestion string     `json:"reflectionQuestion,omitempty"`
	Assets             []JobAsset `json:"assets"`
}

type ClaimRequest struct {
	JobType  database.JobType `json:"jobType"`
	WorkerID string           `json:"workerId"`
}

type CompleteRequest struct {
	OK               bool            `json:"ok"`
	ErrorCode        string          `json:"errorCode,omitempty"`
	ErrorMessage     string          `json:"errorMessage,omitempty"`
	Clips            json.RawMessage `json:"clips,omitempty"`
	SongPickID       string          `json:"songPickId,omitempty"`
	Cut              *CutDTO         `json:"cut,omitempty"`
	RenderStorageKey string          `json:"renderStorageKey,omitempty"`
	RenderURL        string          `json:"renderUrl,omitempty"`
}

type HeartbeatRequest struct {
	WorkerID string `json:"workerId"`
}

func (s *Service) buildCutDraftPayload(
	ctx context.Context,
	principal string,
	momentID, jobID uuid.UUID,
	songPickID string,
	assetIDs []uuid.UUID,
	mode string,
	prior *CutDTO,
	patch *EditorPatch,
) (*CutDraftPayload, error) {
	assets, editURL, err := s.resolveJobAssets(ctx, principal, assetIDs, songPickID)
	if err != nil {
		return nil, err
	}
	return &CutDraftPayload{
		JobType:      string(database.JobCutDraft),
		JobID:        jobID,
		MomentID:     momentID,
		SongPickID:   songPickID,
		EditAudioURL: editURL,
		Mode:         mode,
		Assets:       assets,
		PriorCut:     prior,
		EditorPatch:  patch,
	}, nil
}

func (s *Service) resolveJobAssets(ctx context.Context, principal string, assetIDs []uuid.UUID, songPickID string) ([]JobAsset, string, error) {
	out := make([]JobAsset, len(assetIDs))
	for i, id := range assetIDs {
		out[i] = JobAsset{ID: id, Type: "photo", Order: i}
	}
	editURL := ""

	if s.SkipMedia || s.Media == nil || s.Media.BaseURL == "" {
		return out, editURL, nil
	}

	signed, err := s.Media.SignDownload(ctx, principal, assetIDs, songPickID)
	if err != nil {
		// Fallback: signed GET (sign=true) so Reflection storageUrl still reaches ML.
		metas, gerr := s.Media.GetAssets(ctx, principal, assetIDs)
		if gerr != nil {
			return nil, "", fmt.Errorf("%w: sign/list assets: %v / %v", ErrUpstream, err, gerr)
		}
		byID := map[uuid.UUID]struct {
			Type       string
			DurationMs int
			Width      int
			Height     int
			URL        string
		}{}
		for _, a := range metas {
			dm, w, h := 0, 0, 0
			if a.DurationMs != nil {
				dm = *a.DurationMs
			}
			if a.Width != nil {
				w = *a.Width
			}
			if a.Height != nil {
				h = *a.Height
			}
			byID[a.ID] = struct {
				Type       string
				DurationMs int
				Width      int
				Height     int
				URL        string
			}{a.Type, dm, w, h, a.StorageURL}
		}
		for i, id := range assetIDs {
			if m, ok := byID[id]; ok {
				t := m.Type
				if t == "" {
					t = "photo"
				}
				out[i] = JobAsset{ID: id, Type: t, Order: i, DurationMs: m.DurationMs, Width: m.Width, Height: m.Height, StorageURL: m.URL}
			}
		}
		if n := countMissingStorageURL(out); n > 0 {
			return nil, "", fmt.Errorf(
				"%w: sign-download failed (%v) and GET fallback missing storageUrl for %d/%d Assets",
				ErrUpstream, err, n, len(out),
			)
		}
		return out, editURL, nil
	}

	byID := map[uuid.UUID]JobAsset{}
	for _, a := range signed.Assets {
		dm, w, h := 0, 0, 0
		if a.DurationMs != nil {
			dm = *a.DurationMs
		}
		if a.Width != nil {
			w = *a.Width
		}
		if a.Height != nil {
			h = *a.Height
		}
		t := a.Type
		if t == "" {
			t = "photo"
		}
		byID[a.ID] = JobAsset{ID: a.ID, Type: t, DurationMs: dm, Width: w, Height: h, StorageURL: a.StorageURL}
	}
	for i, id := range assetIDs {
		if a, ok := byID[id]; ok {
			a.Order = i
			out[i] = a
		} else {
			out[i].Order = i
		}
	}
	if n := countMissingStorageURL(out); n > 0 {
		return nil, "", fmt.Errorf(
			"%w: sign-download returned %d/%d Assets without storageUrl (media BaseURL=%q)",
			ErrUpstream, n, len(out), s.Media.BaseURL,
		)
	}
	return out, signed.EditAudioURL, nil
}

func countMissingStorageURL(assets []JobAsset) int {
	n := 0
	for _, a := range assets {
		if strings.TrimSpace(a.StorageURL) == "" {
			n++
		}
	}
	return n
}

func (s *Service) enqueueCutDraft(ctx context.Context, principal string, m *database.Moment, revise bool, patch *EditorPatch) error {
	if err := s.Store.SupersedeActiveJobs(ctx, m.ID, database.JobCutDraft); err != nil {
		return err
	}
	// Also supersede in-flight render — new Cut will enqueue fresh render.
	_ = s.Store.SupersedeActiveJobs(ctx, m.ID, database.JobRender)

	mode := "draft"
	var prior *CutDTO
	if revise {
		mode = "revise"
		if c, err := s.Store.GetCutByMoment(ctx, m.ID); err == nil {
			dto := s.toCutDTO(c)
			prior = &dto
		}
	}

	jobID := uuid.New()
	payload, err := s.buildCutDraftPayload(ctx, principal, m.ID, jobID, m.SongPickID, m.AssetIDs, mode, prior, patch)
	if err != nil {
		return err
	}
	b, _ := json.Marshal(payload)
	job := &database.Job{
		ID:       jobID,
		MomentID: m.ID,
		JobType:  database.JobCutDraft,
		Status:   database.JobQueued,
		Payload:  b,
	}
	if err := s.Store.InsertJob(ctx, job); err != nil {
		if errors.Is(err, database.ErrActiveJobExists) {
			return fmt.Errorf("%w: active cut-draft job", ErrConflict)
		}
		return err
	}
	return nil
}

func (s *Service) enqueueRender(ctx context.Context, principal string, m *database.Moment, cut *database.Cut) error {
	if err := s.Store.SupersedeActiveJobs(ctx, m.ID, database.JobRender); err != nil {
		return err
	}
	jobID := uuid.New()
	assets, editURL, err := s.resolveJobAssets(ctx, principal, m.AssetIDs, m.SongPickID)
	if err != nil {
		return err
	}
	cutDTO := s.toCutDTO(cut)
	rq := ""
	if m.ReflectionQuestion != nil {
		rq = *m.ReflectionQuestion
	}
	payload := RenderPayload{
		JobType:            string(database.JobRender),
		JobID:              jobID,
		MomentID:           m.ID,
		CutVersion:         cut.Version,
		Cut:                cutDTO,
		SongPickID:         m.SongPickID,
		EditAudioURL:       editURL,
		ReflectionQuestion: rq,
		Assets:             assets,
	}
	b, _ := json.Marshal(payload)
	cv := cut.Version
	job := &database.Job{
		ID:         jobID,
		MomentID:   m.ID,
		JobType:    database.JobRender,
		Status:     database.JobQueued,
		Payload:    b,
		CutVersion: &cv,
	}
	if err := s.Store.InsertJob(ctx, job); err != nil {
		if errors.Is(err, database.ErrActiveJobExists) {
			return fmt.Errorf("%w: active render job", ErrConflict)
		}
		return err
	}
	return nil
}

func (s *Service) ClaimJob(ctx context.Context, jobType database.JobType, workerID string) (*database.Job, error) {
	if jobType != database.JobCutDraft && jobType != database.JobRender {
		return nil, fmt.Errorf("%w: unknown jobType", ErrInvalid)
	}
	if workerID == "" {
		return nil, fmt.Errorf("%w: workerId required", ErrInvalid)
	}
	return s.Store.ClaimJob(ctx, jobType, workerID, s.JobLease)
}

func (s *Service) HeartbeatJob(ctx context.Context, jobID uuid.UUID, workerID string) error {
	if workerID == "" {
		return fmt.Errorf("%w: workerId required", ErrInvalid)
	}
	err := s.Store.HeartbeatJob(ctx, jobID, workerID, s.JobLease)
	if errors.Is(err, database.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func (s *Service) CompleteJob(ctx context.Context, jobID uuid.UUID, req CompleteRequest) error {
	job, err := s.Store.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	if job.Status == database.JobSuperseded || job.Status == database.JobSucceeded || job.Status == database.JobFailed {
		// Idempotent no-op for terminal / superseded
		return nil
	}
	if job.Status != database.JobRunning {
		return fmt.Errorf("%w: job not running", ErrConflict)
	}

	m, err := s.Store.GetMoment(ctx, job.MomentID)
	if err != nil {
		return err
	}

	if !req.OK {
		code := req.ErrorCode
		if code == "" {
			code = "JOB_FAILED"
		}
		msg := req.ErrorMessage
		if msg == "" {
			msg = "worker reported failure"
		}
		if err := s.Store.CompleteJobFailure(ctx, jobID, code, msg); err != nil {
			return err
		}
		return s.Store.SetMomentStatus(ctx, m.ID, database.StatusFailed, &code, &msg)
	}

	switch job.JobType {
	case database.JobCutDraft:
		return s.completeCutDraft(ctx, job, m, req)
	case database.JobRender:
		return s.completeRender(ctx, job, m, req)
	default:
		return fmt.Errorf("%w: unknown job type", ErrInvalid)
	}
}

func (s *Service) completeCutDraft(ctx context.Context, job *database.Job, m *database.Moment, req CompleteRequest) error {
	clips := req.Clips
	var captions, markers json.RawMessage
	songPickID := req.SongPickID
	if req.Cut != nil {
		if len(req.Cut.Clips) > 0 {
			clips = req.Cut.Clips
		}
		if len(req.Cut.Captions) > 0 {
			captions = req.Cut.Captions
		}
		if len(req.Cut.Markers) > 0 {
			markers = req.Cut.Markers
		}
		if req.Cut.SongPickID != "" {
			songPickID = req.Cut.SongPickID
		}
	}
	if len(clips) == 0 {
		return fmt.Errorf("%w: clips required on cut-draft success", ErrInvalid)
	}
	if songPickID == "" {
		songPickID = m.SongPickID
	}

	cut, err := s.Store.UpsertCut(ctx, m.ID, songPickID, clips, captions, markers)
	if err != nil {
		return err
	}
	result, _ := json.Marshal(s.toCutDTO(cut))
	if err := s.Store.CompleteJobSuccess(ctx, job.ID, result); err != nil {
		return err
	}
	if err := s.Store.AttachCut(ctx, m.ID, cut.ID); err != nil {
		return err
	}
	m.CutID = &cut.ID
	m.Status = database.StatusRendering
	return s.enqueueRender(ctx, m.UserID, m, cut)
}

func (s *Service) completeRender(ctx context.Context, job *database.Job, m *database.Moment, req CompleteRequest) error {
	// Stale success: cutVersion mismatch → ignore (do not clobber newer ready).
	if job.CutVersion != nil {
		cut, err := s.Store.GetCutByMoment(ctx, m.ID)
		if err == nil && cut.Version != *job.CutVersion {
			_ = s.Store.CompleteJobSuccess(ctx, job.ID, json.RawMessage(`{"stale":true}`))
			return nil
		}
	}
	key := req.RenderStorageKey
	if key == "" {
		return fmt.Errorf("%w: renderStorageKey required", ErrInvalid)
	}
	result, _ := json.Marshal(map[string]string{
		"renderStorageKey": key,
		"renderUrl":        req.RenderURL,
	})
	if err := s.Store.CompleteJobSuccess(ctx, job.ID, result); err != nil {
		return err
	}
	return s.Store.SetMomentReady(ctx, m.ID, key)
}
