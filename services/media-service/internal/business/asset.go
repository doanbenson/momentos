package business

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/doanbenson/momentos/services/media-service/internal/database"
	"github.com/doanbenson/momentos/services/media-service/internal/storage"
	"github.com/google/uuid"
)

const (
	UploadURLTTL   = 15 * time.Minute
	DownloadURLTTL = 15 * time.Minute
	InternalTTL    = 60 * time.Minute
)

// Service manages Asset metadata and upload orchestration against Supabase Storage.
type Service struct {
	store   *database.Store
	storage *storage.Client
}

func NewService(store *database.Store, storageClient *storage.Client) *Service {
	return &Service{store: store, storage: storageClient}
}

type CreateAssetInput struct {
	UserID              string
	Type                database.AssetType
	MimeType            string
	ByteSize            int64
	DurationMs          *int
	Width               *int
	Height              *int
	WantThumbnailUpload bool
}

type CreateAssetResult struct {
	Asset               *database.Asset
	UploadURL           string
	UploadHeaders       map[string]string
	ThumbnailUploadURL  string
	ThumbnailHeaders    map[string]string
}

func (s *Service) CreateAsset(ctx context.Context, in CreateAssetInput) (*CreateAssetResult, error) {
	if in.UserID == "" {
		return nil, fmt.Errorf("%w: user id required", ErrInvalidInput)
	}
	in.MimeType = strings.ToLower(strings.TrimSpace(in.MimeType))
	if !validType(in.Type) {
		return nil, fmt.Errorf("%w: type must be photo|video|voice", ErrInvalidInput)
	}
	if !mimeAllowed(in.Type, in.MimeType) {
		return nil, fmt.Errorf("%w: mime %q not allowed for %s", ErrInvalidInput, in.MimeType, in.Type)
	}
	max := maxBytesFor(in.Type)
	if in.ByteSize <= 0 || in.ByteSize > max {
		return nil, fmt.Errorf("%w: byteSize must be 1..%d for %s", ErrSizeExceeded, max, in.Type)
	}

	since := time.Now().Add(-time.Duration(CreateRateLimitWindow) * time.Second)
	n, err := s.store.CountRecentCreates(ctx, in.UserID, since)
	if err != nil {
		return nil, err
	}
	if n >= CreateRateLimitMax {
		return nil, ErrRateLimited
	}

	id := uuid.New()
	storageKey := fmt.Sprintf("users/%s/assets/%s/original", in.UserID, id.String())
	var thumbKey *string
	wantThumb := in.WantThumbnailUpload || in.Type == database.AssetTypeVideo
	if wantThumb && in.Type != database.AssetTypeVoice {
		k := fmt.Sprintf("users/%s/assets/%s/poster", in.UserID, id.String())
		thumbKey = &k
	}

	asset, err := s.store.CreateAsset(ctx, database.CreateAssetParams{
		ID:           id,
		UserID:       in.UserID,
		Type:         in.Type,
		MimeType:     in.MimeType,
		ByteSize:     in.ByteSize,
		StorageKey:   storageKey,
		ThumbnailKey: thumbKey,
		DurationMs:   in.DurationMs,
		Width:        in.Width,
		Height:       in.Height,
	})
	if err != nil {
		return nil, err
	}

	upload, err := s.storage.SignUpload(ctx, storageKey, UploadURLTTL)
	if err != nil {
		return nil, fmt.Errorf("sign upload url: %w", err)
	}
	if upload.Headers == nil {
		upload.Headers = map[string]string{}
	}
	if _, ok := upload.Headers["Content-Type"]; !ok {
		upload.Headers["Content-Type"] = in.MimeType
	}

	out := &CreateAssetResult{
		Asset:         asset,
		UploadURL:     upload.URL,
		UploadHeaders: upload.Headers,
	}
	if thumbKey != nil {
		tu, err := s.storage.SignUpload(ctx, *thumbKey, UploadURLTTL)
		if err != nil {
			return nil, fmt.Errorf("sign thumbnail upload url: %w", err)
		}
		out.ThumbnailUploadURL = tu.URL
		out.ThumbnailHeaders = tu.Headers
		if out.ThumbnailHeaders == nil {
			out.ThumbnailHeaders = map[string]string{}
		}
		if _, ok := out.ThumbnailHeaders["Content-Type"]; !ok {
			out.ThumbnailHeaders["Content-Type"] = "image/jpeg"
		}
	}
	return out, nil
}

type CompleteAssetInput struct {
	UserID             string
	AssetID            uuid.UUID
	Checksum           *string
	ByteSize           *int64
	DurationMs         *int
	Width              *int
	Height             *int
	ThumbnailUploaded  *bool
}

func (s *Service) CompleteAsset(ctx context.Context, in CompleteAssetInput) (*database.Asset, error) {
	asset, err := s.store.GetAsset(ctx, in.AssetID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if asset.UserID != in.UserID {
		return nil, ErrForbidden
	}
	if asset.Status == database.AssetStatusReady {
		return asset, nil // idempotent
	}
	if asset.Status != database.AssetStatusPending {
		return nil, fmt.Errorf("%w: asset is %s", ErrInvalidStatus, asset.Status)
	}

	info, err := s.storage.StatObject(ctx, asset.StorageKey)
	if err != nil {
		return nil, fmt.Errorf("verify object: %w", err)
	}
	if !info.Exists {
		_ = s.store.MarkAssetFailed(ctx, asset.ID)
		return nil, ErrObjectMissing
	}

	actualSize := asset.ByteSize
	if info.Size > 0 {
		actualSize = info.Size
	}
	if in.ByteSize != nil {
		actualSize = *in.ByteSize
	}
	max := maxBytesFor(asset.Type)
	if actualSize <= 0 || actualSize > max {
		_ = s.store.MarkAssetFailed(ctx, asset.ID)
		return nil, fmt.Errorf("%w: actual size %d", ErrSizeExceeded, actualSize)
	}
	if info.ContentType != "" {
		ct := strings.ToLower(strings.Split(info.ContentType, ";")[0])
		ct = strings.TrimSpace(ct)
		if ct != "" && ct != asset.MimeType && !mimeAllowed(asset.Type, ct) {
			_ = s.store.MarkAssetFailed(ctx, asset.ID)
			return nil, fmt.Errorf("%w: got %s want %s", ErrMimeMismatch, ct, asset.MimeType)
		}
	}

	clearThumb := false
	if asset.ThumbnailKey != nil {
		uploaded := in.ThumbnailUploaded == nil || *in.ThumbnailUploaded
		if uploaded {
			ti, err := s.storage.StatObject(ctx, *asset.ThumbnailKey)
			if err != nil {
				return nil, fmt.Errorf("verify thumbnail: %w", err)
			}
			if !ti.Exists {
				// Client/transform may skip poster — soft-clear key (no ffmpeg).
				clearThumb = true
			}
		} else {
			clearThumb = true
		}
	}

	ready, err := s.store.MarkAssetReady(ctx, database.CompleteAssetParams{
		ID:             asset.ID,
		ByteSize:       actualSize,
		Checksum:       in.Checksum,
		DurationMs:     in.DurationMs,
		Width:          in.Width,
		Height:         in.Height,
		ClearThumbnail: clearThumb,
	})
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return ready, nil
}

type AssetView struct {
	Asset        *database.Asset
	DownloadURL  string
	ThumbnailURL string
}

func (s *Service) GetAsset(ctx context.Context, userID string, id uuid.UUID, sign bool) (*AssetView, error) {
	asset, err := s.store.GetAsset(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if asset.UserID != userID {
		return nil, ErrForbidden
	}
	return s.viewAsset(ctx, asset, sign, DownloadURLTTL)
}

func (s *Service) ListAssets(ctx context.Context, userID string, ids []uuid.UUID, sign bool) ([]AssetView, error) {
	assets, err := s.store.ListAssetsByIDs(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]AssetView, 0, len(assets))
	for i := range assets {
		v, err := s.viewAsset(ctx, &assets[i], sign, DownloadURLTTL)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, nil
}

func (s *Service) DeleteAsset(ctx context.Context, userID string, id uuid.UUID) error {
	asset, err := s.store.DeleteAsset(ctx, id, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	_ = s.storage.RemoveObject(ctx, asset.StorageKey)
	if asset.ThumbnailKey != nil {
		_ = s.storage.RemoveObject(ctx, *asset.ThumbnailKey)
	}
	return nil
}

type SignDownloadInput struct {
	AssetIDs   []uuid.UUID
	SongPickID string
}

type SignedAsset struct {
	AssetID     uuid.UUID           `json:"assetId"`
	Type        database.AssetType  `json:"type"`
	MimeType    string              `json:"mimeType"`
	Status      database.AssetStatus `json:"status"`
	DownloadURL string              `json:"downloadUrl"`
	ThumbnailURL string             `json:"thumbnailUrl,omitempty"`
	DurationMs  *int                `json:"durationMs,omitempty"`
	Width       *int                `json:"width,omitempty"`
	Height      *int                `json:"height,omitempty"`
	StorageKey  string              `json:"storageKey"`
}

type SignDownloadResult struct {
	Assets           []SignedAsset `json:"assets"`
	EditAudioURL     string        `json:"editAudioUrl,omitempty"`
	EditAudioMime    string        `json:"editAudioMime,omitempty"`
	EditAudioKey     string        `json:"editAudioStorageKey,omitempty"`
	SongPickID       string        `json:"songPickId,omitempty"`
}

// SignDownload is the internal helper for ml / render (no owner gate).
func (s *Service) SignDownload(ctx context.Context, in SignDownloadInput) (*SignDownloadResult, error) {
	assets, err := s.store.ListAssetsByIDsAny(ctx, in.AssetIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]database.Asset, len(assets))
	for _, a := range assets {
		byID[a.ID] = a
	}

	out := &SignDownloadResult{Assets: make([]SignedAsset, 0, len(in.AssetIDs))}
	for _, id := range in.AssetIDs {
		a, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("%w: asset %s", ErrNotFound, id)
		}
		if a.Status != database.AssetStatusReady {
			return nil, fmt.Errorf("%w: asset %s is %s", ErrInvalidStatus, id, a.Status)
		}
		url, err := s.storage.SignDownload(ctx, a.StorageKey, InternalTTL)
		if err != nil {
			return nil, err
		}
		sa := SignedAsset{
			AssetID:     a.ID,
			Type:        a.Type,
			MimeType:    a.MimeType,
			Status:      a.Status,
			DownloadURL: url,
			DurationMs:  a.DurationMs,
			Width:       a.Width,
			Height:      a.Height,
			StorageKey:  a.StorageKey,
		}
		if a.ThumbnailKey != nil {
			tu, err := s.storage.SignDownload(ctx, *a.ThumbnailKey, InternalTTL)
			if err == nil {
				sa.ThumbnailURL = tu
			}
		}
		out.Assets = append(out.Assets, sa)
	}

	if in.SongPickID != "" {
		sp, err := s.store.GetSongPick(ctx, in.SongPickID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, fmt.Errorf("%w: song pick", ErrNotFound)
			}
			return nil, err
		}
		url, err := s.storage.SignDownload(ctx, sp.EditAudioStorageKey, InternalTTL)
		if err != nil {
			return nil, err
		}
		out.SongPickID = sp.ID
		out.EditAudioURL = url
		out.EditAudioMime = sp.EditAudioMime
		out.EditAudioKey = sp.EditAudioStorageKey
	}
	return out, nil
}

func (s *Service) viewAsset(ctx context.Context, asset *database.Asset, sign bool, ttl time.Duration) (*AssetView, error) {
	v := &AssetView{Asset: asset}
	if !sign || asset.Status != database.AssetStatusReady {
		return v, nil
	}
	url, err := s.storage.SignDownload(ctx, asset.StorageKey, ttl)
	if err != nil {
		return nil, err
	}
	v.DownloadURL = url
	if asset.ThumbnailKey != nil {
		tu, err := s.storage.SignDownload(ctx, *asset.ThumbnailKey, ttl)
		if err == nil {
			v.ThumbnailURL = tu
		}
	}
	return v, nil
}

func validType(t database.AssetType) bool {
	switch t {
	case database.AssetTypePhoto, database.AssetTypeVideo, database.AssetTypeVoice:
		return true
	default:
		return false
	}
}
