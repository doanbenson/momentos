package business

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/doanbenson/momentos/services/moment-service/internal/database"
	"github.com/doanbenson/momentos/services/moment-service/internal/media"
)

const DefaultSongPickID = "live-a-life-you-will-remember"
const DefaultSongPickTitle = "Live A Life You Will Remember"

var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
	ErrInvalid          = errors.New("invalid request")
	ErrUpstream         = errors.New("upstream error")
	ErrIdempotencyReuse = errors.New("idempotency key conflict")
)

type Service struct {
	Store         Store
	Media         *media.Client
	SongPickID    string
	SongPickTitle string
	JobLease      time.Duration
	SkipMedia     bool // local/dev when media-service is not up
}

func NewService(store *database.Store, mediaClient *media.Client) *Service {
	songID := os.Getenv("SONG_PICK_ID")
	if songID == "" {
		songID = DefaultSongPickID
	}
	title := os.Getenv("SONG_PICK_TITLE")
	if title == "" {
		title = DefaultSongPickTitle
	}
	lease := 5 * time.Minute
	if v := os.Getenv("JOB_LEASE_SECONDS"); v != "" {
		if secs, err := time.ParseDuration(v + "s"); err == nil {
			lease = secs
		}
	}
	return &Service{
		Store:         store,
		Media:         mediaClient,
		SongPickID:    songID,
		SongPickTitle: title,
		JobLease:      lease,
		SkipMedia:     os.Getenv("MEDIA_SKIP_VALIDATE") == "true",
	}
}

type CreateMomentInput struct {
	Principal          string
	AssetIDs           []uuid.UUID
	ReflectionAssetID  uuid.UUID
	ReflectionQuestion string
	SongPickID         string
	IdempotencyKey     string
	RawBody            []byte
}

type MomentDTO struct {
	ID                 uuid.UUID             `json:"id"`
	Status             database.MomentStatus `json:"status"`
	SongPick           SongPickDTO           `json:"songPick"`
	AssetIDs           []uuid.UUID           `json:"assetIds"`
	ReflectionAssetID  *uuid.UUID            `json:"reflectionAssetId,omitempty"`
	ReflectionQuestion *string               `json:"reflectionQuestion,omitempty"`
	CutID              *uuid.UUID            `json:"cutId,omitempty"`
	RenderStorageKey   *string               `json:"renderStorageKey,omitempty"`
	RenderURL          *string               `json:"renderUrl,omitempty"`
	ErrorCode          *string               `json:"errorCode,omitempty"`
	ErrorMessage       *string               `json:"errorMessage,omitempty"`
}

type SongPickDTO struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	PlatformSongID *string `json:"platformSongId,omitempty"`
}

type CutDTO struct {
	ID         uuid.UUID       `json:"id"`
	MomentID   uuid.UUID       `json:"momentId"`
	Version    int             `json:"version"`
	SongPickID string          `json:"songPickId,omitempty"`
	Clips      json.RawMessage `json:"clips"`
	Captions   json.RawMessage `json:"captions,omitempty"`
	Markers    json.RawMessage `json:"markers,omitempty"`
}

type ClipDTO struct {
	AssetID      uuid.UUID `json:"assetId"`
	Order        int       `json:"order"`
	StartMs      *int      `json:"startMs,omitempty"`
	EndMs        *int      `json:"endMs,omitempty"`
	Role         string    `json:"role,omitempty"`
	Act          *int      `json:"act,omitempty"`
	MotionPreset *string   `json:"motionPreset,omitempty"`
	VisualStyle  *string   `json:"visualStyle,omitempty"`
	TransitionIn *string   `json:"transitionIn,omitempty"`
	SourceInMs   *int      `json:"sourceInMs,omitempty"`
	SourceOutMs  *int      `json:"sourceOutMs,omitempty"`
}

// ClipTrim is one Editor source-window edit. sourceInMs is inclusive, sourceOutMs exclusive.
type ClipTrim struct {
	AssetID     uuid.UUID `json:"assetId"`
	SourceInMs  int       `json:"sourceInMs"`
	SourceOutMs int       `json:"sourceOutMs"`
}

type EditorPatch struct {
	ReorderAssetIDs []uuid.UUID `json:"reorderAssetIds"`
	Replace         *struct {
		FromAssetID uuid.UUID `json:"fromAssetId"`
		ToAssetID   uuid.UUID `json:"toAssetId"`
	} `json:"replace"`
	RemoveAssetID *uuid.UUID `json:"removeAssetId"`
	Trims         []ClipTrim `json:"trims"`
	ReDraft       bool       `json:"reDraft"`
	ReRender      bool       `json:"reRender"`
}

// MomentListItem is one row of GET /v1/moments. renderUrl is omitted:
// the moment row stores a storage key, and signing each row would block the list on media.
type MomentListItem struct {
	ID         uuid.UUID             `json:"id"`
	Status     database.MomentStatus `json:"status"`
	CreatedAt  time.Time             `json:"createdAt"`
	SongPickID string                `json:"songPickId"`
	RenderURL  *string               `json:"renderUrl,omitempty"`
}

func (s *Service) toMomentDTO(m *database.Moment) MomentDTO {
	dto := MomentDTO{
		ID:     m.ID,
		Status: m.Status,
		SongPick: SongPickDTO{
			ID:    m.SongPickID,
			Title: s.songTitle(m.SongPickID),
		},
		AssetIDs:         m.AssetIDs,
		CutID:            m.CutID,
		RenderStorageKey: m.RenderStorageKey,
		ErrorCode:        m.ErrorCode,
		ErrorMessage:     m.ErrorMessage,
	}
	if m.ReflectionAssetID != nil {
		dto.ReflectionAssetID = m.ReflectionAssetID
	}
	if m.ReflectionQuestion != nil && *m.ReflectionQuestion != "" {
		dto.ReflectionQuestion = m.ReflectionQuestion
	}
	return dto
}

func (s *Service) songTitle(id string) string {
	if id == s.SongPickID || id == DefaultSongPickID {
		return s.SongPickTitle
	}
	return id
}

func (s *Service) toCutDTO(c *database.Cut) CutDTO {
	clips := c.Clips
	if len(clips) == 0 {
		clips = json.RawMessage("[]")
	}
	return CutDTO{
		ID:         c.ID,
		MomentID:   c.MomentID,
		Version:    c.Version,
		SongPickID: c.SongPickID,
		Clips:      clips,
		Captions:   c.Captions,
		Markers:    c.Markers,
	}
}

func (s *Service) CreateMoment(ctx context.Context, in CreateMomentInput) (*MomentDTO, error) {
	if in.Principal == "" {
		return nil, ErrUnauthorized
	}
	if err := ValidateCreateMomentContract(in); err != nil {
		return nil, err
	}
	songPickID := in.SongPickID
	if songPickID == "" {
		songPickID = s.SongPickID
	}

	reqHash := hashBody(in.RawBody)
	if in.IdempotencyKey != "" {
		rec, err := s.Store.GetIdempotency(ctx, in.Principal, in.IdempotencyKey)
		if err == nil {
			if rec.RequestHash != reqHash {
				return nil, ErrIdempotencyReuse
			}
			m, err := s.Store.GetMoment(ctx, rec.MomentID)
			if err != nil {
				return nil, err
			}
			dto := s.toMomentDTO(m)
			return &dto, nil
		}
		if !errors.Is(err, database.ErrNotFound) {
			return nil, err
		}
	}

	if err := s.validateAssetsReady(ctx, in.Principal, in.AssetIDs); err != nil {
		return nil, err
	}
	if err := s.validateReflectionAsset(ctx, in.Principal, in.ReflectionAssetID); err != nil {
		return nil, err
	}

	momentID := uuid.New()
	jobID := uuid.New()

	payload, err := s.buildCutDraftPayload(ctx, in.Principal, momentID, jobID, songPickID, in.AssetIDs, "draft", nil, nil)
	if err != nil {
		return nil, err
	}
	payload.ReflectionAssetID = in.ReflectionAssetID
	payload.ReflectionQuestion = in.ReflectionQuestion
	payloadBytes, _ := json.Marshal(payload)

	rq := in.ReflectionQuestion
	ra := in.ReflectionAssetID
	m := &database.Moment{
		ID:                 momentID,
		UserID:             in.Principal,
		Status:             database.StatusDrafting,
		SongPickID:         songPickID,
		AssetIDs:           in.AssetIDs,
		ReflectionAssetID:  &ra,
		ReflectionQuestion: &rq,
	}
	job := &database.Job{
		ID:       jobID,
		MomentID: momentID,
		JobType:  database.JobCutDraft,
		Status:   database.JobQueued,
		Payload:  payloadBytes,
	}
	if in.IdempotencyKey != "" {
		job.IdempotencyKey = &in.IdempotencyKey
	}

	var idem *database.IdempotencyRecord
	if in.IdempotencyKey != "" {
		idem = &database.IdempotencyRecord{
			UserID:      in.Principal,
			Key:         in.IdempotencyKey,
			RequestHash: reqHash,
			MomentID:    momentID,
		}
	}

	if err := s.Store.CreateMomentWithJob(ctx, m, job, idem); err != nil {
		return nil, err
	}
	dto := s.toMomentDTO(m)
	return &dto, nil
}

func (s *Service) ListMoments(ctx context.Context, principal string) ([]MomentListItem, error) {
	if principal == "" {
		return nil, ErrUnauthorized
	}
	ms, err := s.Store.ListMomentsForUser(ctx, principal)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(ms, func(i, j int) bool {
		return ms[i].CreatedAt.After(ms[j].CreatedAt)
	})
	items := make([]MomentListItem, 0, len(ms))
	for _, m := range ms {
		items = append(items, MomentListItem{
			ID:         m.ID,
			Status:     m.Status,
			CreatedAt:  m.CreatedAt,
			SongPickID: m.SongPickID,
		})
	}
	return items, nil
}

func (s *Service) GetMoment(ctx context.Context, principal string, id uuid.UUID) (*MomentDTO, error) {
	m, err := s.Store.GetMomentForUser(ctx, id, principal)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	dto := s.toMomentDTO(m)
	if m.Status == database.StatusReady && m.RenderStorageKey != nil && *m.RenderStorageKey != "" &&
		s.Media != nil && s.Media.BaseURL != "" {
		if url, err := s.Media.SignRenderURL(ctx, principal, *m.RenderStorageKey); err == nil && url != "" {
			dto.RenderURL = &url
		}
	}
	return &dto, nil
}

func (s *Service) GetCut(ctx context.Context, principal string, momentID uuid.UUID) (*CutDTO, error) {
	if _, err := s.Store.GetMomentForUser(ctx, momentID, principal); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c, err := s.Store.GetCutByMoment(ctx, momentID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	dto := s.toCutDTO(c)
	return &dto, nil
}

func (s *Service) ApplyEditorPatch(ctx context.Context, principal string, momentID uuid.UUID, patch EditorPatch) (*CutDTO, error) {
	m, err := s.Store.GetMomentForUser(ctx, momentID, principal)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	cut, cutErr := s.Store.GetCutByMoment(ctx, momentID)
	if cutErr != nil && !errors.Is(cutErr, database.ErrNotFound) {
		return nil, cutErr
	}
	hasCut := cutErr == nil

	if hasCut && clipEditRequested(patch) {
		// Edit the stored Cut in place. reDraft is the only path that enqueues cut-draft.
		clips, captions, assetIDs, editErr := applyCutEdits(cut.Clips, cut.Captions, patch)
		if editErr != nil {
			return nil, editErr
		}
		if patch.Replace != nil {
			if err := s.validateAssetsReady(ctx, principal, []uuid.UUID{patch.Replace.ToAssetID}); err != nil {
				return nil, err
			}
		}
		updated, err := s.Store.UpsertCut(ctx, momentID, cut.SongPickID, clips, captions, cut.Markers)
		if err != nil {
			return nil, err
		}
		if err := s.Store.UpdateMomentAssetIDs(ctx, momentID, assetIDs); err != nil {
			return nil, err
		}
		m.AssetIDs = assetIDs
		cut = updated
	} else if !hasCut {
		if len(patch.Trims) > 0 {
			return nil, fmt.Errorf("%w: no Cut to trim", ErrInvalid)
		}
		assetIDs := append([]uuid.UUID(nil), m.AssetIDs...)
		changed := false

		if len(patch.ReorderAssetIDs) > 0 {
			if err := validateReorder(assetIDs, patch.ReorderAssetIDs); err != nil {
				return nil, err
			}
			assetIDs = append([]uuid.UUID(nil), patch.ReorderAssetIDs...)
			changed = true
		}
		if patch.Replace != nil {
			found := false
			for i, id := range assetIDs {
				if id == patch.Replace.FromAssetID {
					assetIDs[i] = patch.Replace.ToAssetID
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("%w: fromAssetId not in moment", ErrInvalid)
			}
			if err := s.validateAssetsReady(ctx, principal, []uuid.UUID{patch.Replace.ToAssetID}); err != nil {
				return nil, err
			}
			changed = true
		}
		if patch.RemoveAssetID != nil {
			next := make([]uuid.UUID, 0, len(assetIDs))
			for _, id := range assetIDs {
				if id != *patch.RemoveAssetID {
					next = append(next, id)
				}
			}
			if len(next) == len(assetIDs) {
				return nil, fmt.Errorf("%w: removeAssetId not in moment", ErrInvalid)
			}
			if len(next) == 0 {
				return nil, fmt.Errorf("%w: cannot remove last asset", ErrInvalid)
			}
			assetIDs = next
			changed = true
		}
		if changed {
			if err := s.Store.UpdateMomentAssetIDs(ctx, momentID, assetIDs); err != nil {
				return nil, err
			}
			m.AssetIDs = assetIDs
		}
	}

	// both flags → reDraft only; render enqueued after cut-draft completes.
	// trim/reorder/replace/remove with reRender must not enqueue cut-draft.
	if patch.ReDraft {
		if err := s.enqueueCutDraft(ctx, principal, m, true, &patch); err != nil {
			return nil, err
		}
		_ = s.Store.SetMomentStatus(ctx, momentID, database.StatusDrafting, nil, nil)
	} else if patch.ReRender {
		if !hasCut {
			return nil, fmt.Errorf("%w: no Cut to re-render", ErrConflict)
		}
		if err := s.enqueueRender(ctx, principal, m, cut); err != nil {
			return nil, err
		}
		_ = s.Store.SetMomentStatus(ctx, momentID, database.StatusRendering, nil, nil)
	}

	if !hasCut {
		empty := &CutDTO{
			ID:         uuid.Nil,
			MomentID:   momentID,
			Version:    0,
			SongPickID: m.SongPickID,
			Clips:      json.RawMessage("[]"),
		}
		return empty, nil
	}
	dto := s.toCutDTO(cut)
	return &dto, nil
}

func validateReorder(current, next []uuid.UUID) error {
	if len(current) != len(next) {
		return fmt.Errorf("%w: reorderAssetIds length mismatch", ErrInvalid)
	}
	a := append([]uuid.UUID(nil), current...)
	b := append([]uuid.UUID(nil), next...)
	sort.Slice(a, func(i, j int) bool { return a[i].String() < a[j].String() })
	sort.Slice(b, func(i, j int) bool { return b[i].String() < b[j].String() })
	for i := range a {
		if a[i] != b[i] {
			return fmt.Errorf("%w: reorderAssetIds must be a permutation", ErrInvalid)
		}
	}
	return nil
}

func (s *Service) validateAssetsReady(ctx context.Context, principal string, ids []uuid.UUID) error {
	if s.SkipMedia || s.Media == nil || s.Media.BaseURL == "" {
		return nil
	}
	assets, err := s.Media.GetAssets(ctx, principal, ids)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	byID := map[uuid.UUID]media.AssetSummary{}
	for _, a := range assets {
		byID[a.ID] = a
	}
	for _, id := range ids {
		a, ok := byID[id]
		if !ok {
			return fmt.Errorf("%w: asset %s not found", ErrInvalid, id)
		}
		if a.UserID != "" && a.UserID != principal {
			return fmt.Errorf("%w: asset %s not owned", ErrForbidden, id)
		}
		if a.Status != "" && a.Status != "ready" {
			return fmt.Errorf("%w: asset %s status=%s (want ready)", ErrInvalid, id, a.Status)
		}
	}
	return nil
}

func (s *Service) validateReflectionAsset(ctx context.Context, principal string, reflectionID uuid.UUID) error {
	if s.SkipMedia || s.Media == nil || s.Media.BaseURL == "" {
		return nil
	}
	assets, err := s.Media.GetAssets(ctx, principal, []uuid.UUID{reflectionID})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	if len(assets) == 0 {
		return fmt.Errorf("%w: reflection asset %s not found", ErrInvalid, reflectionID)
	}
	return validateReflectionAssetType(assets[0].Type)
}

func hashBody(b []byte) string {
	sum := sha256.Sum256(bytes.TrimSpace(b))
	return hex.EncodeToString(sum[:])
}
