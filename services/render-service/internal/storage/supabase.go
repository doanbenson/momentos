package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Client uploads/signs Render objects in Supabase Storage (service role).
type Client struct {
	baseURL    string
	serviceKey string
	bucket     string
	http       *http.Client
}

func NewFromEnv() *Client {
	base := strings.TrimRight(os.Getenv("SUPABASE_URL"), "/")
	key := os.Getenv("SUPABASE_SECRET_KEY")
	if base == "" || key == "" {
		return nil
	}
	bucket := os.Getenv("MEDIA_STORAGE_BUCKET")
	if bucket == "" {
		bucket = "momentos-media"
	}
	return &Client{
		baseURL:    base,
		serviceKey: key,
		bucket:     bucket,
		http:       &http.Client{Timeout: 120 * time.Second},
	}
}

// UploadFile PUTs localPath to objectKey and returns a signed GET URL.
func (c *Client) UploadFile(ctx context.Context, localPath, objectKey string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("storage client unset")
	}
	raw, err := os.ReadFile(localPath)
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("/storage/v1/object/%s/%s",
		url.PathEscape(c.bucket), escapeKey(objectKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("apikey", c.serviceKey)
	req.Header.Set("Content-Type", "video/mp4")
	req.Header.Set("x-upsert", "true")
	res, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("upload status %d: %s", res.StatusCode, truncate(string(body), 200))
	}
	return c.SignDownload(ctx, objectKey, time.Hour)
}

// SignDownload returns a short-lived GET URL for objectKey.
func (c *Client) SignDownload(ctx context.Context, objectKey string, expiresIn time.Duration) (string, error) {
	path := fmt.Sprintf("/storage/v1/object/sign/%s/%s",
		url.PathEscape(c.bucket), escapeKey(objectKey))
	payload, _ := json.Marshal(map[string]any{"expiresIn": int(expiresIn.Seconds())})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("apikey", c.serviceKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("sign status %d: %s", res.StatusCode, truncate(string(b), 200))
	}
	var resp struct {
		SignedURL string `json:"signedURL"`
		SignedUrl string `json:"signedUrl"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return "", err
	}
	signed := resp.SignedURL
	if signed == "" {
		signed = resp.SignedUrl
	}
	if signed == "" {
		return "", fmt.Errorf("sign download: empty url")
	}
	if !strings.HasPrefix(signed, "http") {
		signed = c.baseURL + "/storage/v1" + signed
	}
	return signed, nil
}

func escapeKey(key string) string {
	parts := strings.Split(key, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
