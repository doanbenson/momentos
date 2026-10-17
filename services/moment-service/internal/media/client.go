package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Client talks to media-service for Asset ownership checks and signed URLs.
type Client struct {
	BaseURL       string
	InternalToken string
	HTTP          *http.Client
}

func NewClient(baseURL, internalToken string) *Client {
	return &Client{
		BaseURL:       strings.TrimRight(baseURL, "/"),
		InternalToken: internalToken,
		HTTP:          &http.Client{Timeout: 15 * time.Second},
	}
}

type AssetSummary struct {
	ID         uuid.UUID `json:"id"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	UserID     string    `json:"userId,omitempty"`
	DurationMs *int      `json:"durationMs,omitempty"`
	Width      *int      `json:"width,omitempty"`
	Height     *int      `json:"height,omitempty"`
	StorageURL string    `json:"downloadUrl,omitempty"`
	ThumbURL   string    `json:"thumbnailUrl,omitempty"`
}

type SignDownloadRequest struct {
	AssetIDs   []uuid.UUID `json:"assetIds"`
	SongPickID string      `json:"songPickId,omitempty"`
	UserID     string      `json:"userId,omitempty"`
}

type SignedAsset struct {
	ID         uuid.UUID `json:"id"`
	Type       string    `json:"type"`
	DurationMs *int      `json:"durationMs,omitempty"`
	Width      *int      `json:"width,omitempty"`
	Height     *int      `json:"height,omitempty"`
	StorageURL string    `json:"storageUrl"`
}

type SignDownloadResponse struct {
	Assets       []SignedAsset `json:"assets"`
	EditAudioURL string        `json:"editAudioUrl,omitempty"`
}

type SongPick struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	PlatformSongID *string `json:"platformSongId,omitempty"`
	EditAudioURL   string  `json:"editAudioUrl,omitempty"`
}

func (c *Client) GetAssets(ctx context.Context, principal string, ids []uuid.UUID) ([]AssetSummary, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := url.Values{}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = id.String()
	}
	q.Set("ids", strings.Join(parts, ","))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v1/assets?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	c.auth(req, principal)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("media GET assets: %s: %s", res.Status, truncate(body))
	}
	var out struct {
		Assets []AssetSummary `json:"assets"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		// allow bare array
		if err2 := json.Unmarshal(body, &out.Assets); err2 != nil {
			return nil, fmt.Errorf("decode assets: %w", err)
		}
	}
	return out.Assets, nil
}

func (c *Client) SignDownload(ctx context.Context, principal string, assetIDs []uuid.UUID, songPickID string) (*SignDownloadResponse, error) {
	payload, _ := json.Marshal(SignDownloadRequest{
		AssetIDs:   assetIDs,
		SongPickID: songPickID,
		UserID:     principal,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/internal/assets/sign-download", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req, principal)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("media sign-download: %s: %s", res.Status, truncate(body))
	}
	var out SignDownloadResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSongPick(ctx context.Context, principal, id string) (*SongPick, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v1/song-picks/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}
	c.auth(req, principal)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("media GET song-pick: %s: %s", res.Status, truncate(body))
	}
	var out SongPick
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SignRenderURL(ctx context.Context, principal, storageKey string) (string, error) {
	// Prefer song-pick style helper; media may expose signed object URLs via sign-download.
	// Until media lands a dedicated render-sign route, return empty and let gateway omit.
	_ = ctx
	_ = principal
	_ = storageKey
	return "", nil
}

func (c *Client) auth(req *http.Request, principal string) {
	if c.InternalToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.InternalToken)
		req.Header.Set("X-Momentos-Internal-Token", c.InternalToken)
	}
	if principal != "" {
		req.Header.Set("X-Momentos-Principal", principal)
	}
}

func truncate(b []byte) string {
	s := string(b)
	if len(s) > 200 {
		return s[:200]
	}
	return s
}
