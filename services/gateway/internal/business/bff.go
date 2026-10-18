package business

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
)

// BFF orchestrates calls to moment-service, media-service, and render-service.
// No domain persistence lives here.
type BFF struct {
	Moment *PeerClient
	Media  *PeerClient
	Render *PeerClient

	SongPickID    string
	SongPickTitle string
}

func NewBFF(momentURL, mediaURL, renderURL, internalToken string) *BFF {
	songID := os.Getenv("SONG_PICK_ID")
	if songID == "" {
		songID = "live-a-life-you-will-remember"
	}
	title := os.Getenv("SONG_PICK_TITLE")
	if title == "" {
		title = "Live A Life You Will Remember"
	}
	return &BFF{
		Moment:        NewPeer(momentURL, internalToken),
		Media:         NewPeer(mediaURL, internalToken),
		Render:        NewPeer(renderURL, internalToken),
		SongPickID:    songID,
		SongPickTitle: title,
	}
}

type MomentLean struct {
	ID                 uuid.UUID   `json:"id"`
	Status             string      `json:"status"`
	SongPick           SongPick    `json:"songPick"`
	AssetIDs           []uuid.UUID `json:"assetIds"`
	ReflectionAssetID  *uuid.UUID  `json:"reflectionAssetId,omitempty"`
	ReflectionQuestion *string     `json:"reflectionQuestion,omitempty"`
	CutID              *uuid.UUID  `json:"cutId,omitempty"`
	RenderStorageKey   *string     `json:"renderStorageKey,omitempty"`
	RenderURL          *string     `json:"renderUrl,omitempty"`
	ErrorCode          *string     `json:"errorCode,omitempty"`
	ErrorMessage       *string     `json:"errorMessage,omitempty"`
}

type SongPick struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	PlatformSongID *string `json:"platformSongId,omitempty"`
}

type Cut struct {
	ID         uuid.UUID       `json:"id"`
	MomentID   uuid.UUID       `json:"momentId"`
	Version    int             `json:"version"`
	SongPickID string          `json:"songPickId,omitempty"`
	Clips      json.RawMessage `json:"clips"`
	Captions   json.RawMessage `json:"captions,omitempty"`
	Markers    json.RawMessage `json:"markers,omitempty"`
}

type AssetSummary struct {
	ID           uuid.UUID `json:"id"`
	Type         string    `json:"type"`
	ThumbnailURL *string   `json:"thumbnailUrl,omitempty"`
	DurationMs   *int      `json:"durationMs,omitempty"`
}

// MomentAggregate is the BFF GET shape (OpenAPI Moment + optional nesteds).
type MomentAggregate struct {
	ID                 uuid.UUID      `json:"id"`
	Status             string         `json:"status"`
	SongPick           SongPick       `json:"songPick"`
	AssetIDs           []uuid.UUID    `json:"assetIds"`
	ReflectionAssetID  *uuid.UUID     `json:"reflectionAssetId,omitempty"`
	ReflectionQuestion *string        `json:"reflectionQuestion,omitempty"`
	Assets             []AssetSummary `json:"assets,omitempty"`
	CutID              *uuid.UUID     `json:"cutId,omitempty"`
	Cut                *Cut           `json:"cut,omitempty"`
	RenderURL          *string        `json:"renderUrl,omitempty"`
	ErrorCode          *string        `json:"errorCode,omitempty"`
	ErrorMessage       *string        `json:"errorMessage,omitempty"`
}

func (b *BFF) ProxyCreateMoment(ctx context.Context, principal string, body []byte, idemKey string) (int, []byte, error) {
	extra := http.Header{}
	if idemKey != "" {
		extra.Set("Idempotency-Key", idemKey)
	}
	return b.Moment.Do(ctx, http.MethodPost, "/v1/moments", principal, body, extra)
}

func (b *BFF) ProxyGetCut(ctx context.Context, principal, momentID string) (int, []byte, error) {
	return b.Moment.Do(ctx, http.MethodGet, "/v1/moments/"+momentID+"/cut", principal, nil, nil)
}

func (b *BFF) ProxyPatchCut(ctx context.Context, principal, momentID string, body []byte) (int, []byte, error) {
	return b.Moment.Do(ctx, http.MethodPatch, "/v1/moments/"+momentID+"/cut", principal, body, nil)
}

// --- media-service thin proxies (Week Intake upload path) ---

func (b *BFF) ProxyMedia(ctx context.Context, method, path, principal string, body []byte, query string) (int, []byte, error) {
	if query != "" {
		path = path + "?" + query
	}
	return b.Media.Do(ctx, method, path, principal, body, nil)
}

func (b *BFF) GetMomentAggregate(ctx context.Context, principal, momentID string) (*MomentAggregate, error) {
	var lean MomentLean
	if err := b.Moment.JSON(ctx, http.MethodGet, "/v1/moments/"+momentID, principal, nil, &lean); err != nil {
		return nil, err
	}

	agg := &MomentAggregate{
		ID:                 lean.ID,
		Status:             lean.Status,
		SongPick:           lean.SongPick,
		AssetIDs:           lean.AssetIDs,
		ReflectionAssetID:  lean.ReflectionAssetID,
		ReflectionQuestion: lean.ReflectionQuestion,
		CutID:              lean.CutID,
		RenderURL:          lean.RenderURL,
		ErrorCode:          lean.ErrorCode,
		ErrorMessage:       lean.ErrorMessage,
	}
	if agg.SongPick.Title == "" {
		agg.SongPick.Title = b.SongPickTitle
	}
	if agg.SongPick.ID == "" {
		agg.SongPick.ID = b.SongPickID
	}

	// Enrich Song Pick from media when available.
	if b.Media.BaseURL != "" {
		var sp SongPick
		if err := b.Media.JSON(ctx, http.MethodGet, "/v1/song-picks/"+lean.SongPick.ID, principal, nil, &sp); err == nil && sp.ID != "" {
			agg.SongPick = sp
		}
	}

	if len(lean.AssetIDs) > 0 && b.Media.BaseURL != "" {
		if assets, err := b.fetchAssets(ctx, principal, lean.AssetIDs); err == nil {
			agg.Assets = assets
		}
	}

	if lean.CutID != nil {
		var cut Cut
		if err := b.Moment.JSON(ctx, http.MethodGet, "/v1/moments/"+momentID+"/cut", principal, nil, &cut); err == nil {
			agg.Cut = &cut
		}
	}

	if lean.Status == "ready" {
		if url, err := b.resolveRenderURL(ctx, principal, lean); err == nil && url != "" {
			agg.RenderURL = &url
		}
	}

	return agg, nil
}

func (b *BFF) fetchAssets(ctx context.Context, principal string, ids []uuid.UUID) ([]AssetSummary, error) {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = id.String()
	}
	path := "/v1/assets?ids=" + joinCSV(parts)
	var wrap struct {
		Assets []struct {
			ID           uuid.UUID `json:"id"`
			Type         string    `json:"type"`
			ThumbnailURL *string   `json:"thumbnailUrl"`
			DurationMs   *int      `json:"durationMs"`
		} `json:"assets"`
	}
	status, raw, err := b.Media.Do(ctx, http.MethodGet, path, principal, nil, nil)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, &HTTPError{Status: status, Body: raw}
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, err
	}
	out := make([]AssetSummary, 0, len(wrap.Assets))
	for _, a := range wrap.Assets {
		out = append(out, AssetSummary{
			ID:           a.ID,
			Type:         a.Type,
			ThumbnailURL: a.ThumbnailURL,
			DurationMs:   a.DurationMs,
		})
	}
	return out, nil
}

func (b *BFF) resolveRenderURL(ctx context.Context, principal string, lean MomentLean) (string, error) {
	if lean.RenderURL != nil && *lean.RenderURL != "" {
		return *lean.RenderURL, nil
	}
	if b.Media.BaseURL == "" || lean.RenderStorageKey == nil || *lean.RenderStorageKey == "" {
		return "", nil
	}
	payload := map[string]string{"storageKey": *lean.RenderStorageKey}
	status, raw, err := b.Media.Do(ctx, http.MethodPost, "/v1/internal/objects/sign-download", principal, mustJSON(payload), nil)
	if err != nil || status >= 300 {
		return "", nil
	}
	var out struct {
		DownloadURL string `json:"downloadUrl"`
	}
	_ = json.Unmarshal(raw, &out)
	return out.DownloadURL, nil
}

func joinCSV(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += "," + parts[i]
	}
	return out
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Errorf("json: %w", err))
	}
	return b
}
