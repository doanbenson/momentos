package storage

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
)

// Client mints Supabase Storage signed upload/download URLs with the service role.
// Never expose the service-role key to mobile.
type Client struct {
	baseURL    string // e.g. https://xxx.supabase.co
	serviceKey string
	bucket     string
	http       *http.Client
}

func New(supabaseURL, serviceRoleKey, bucket string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(supabaseURL, "/"),
		serviceKey: serviceRoleKey,
		bucket:     bucket,
		http:       &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Bucket() string { return c.bucket }

type SignedUpload struct {
	URL     string
	Headers map[string]string
	Token   string
}

// SignUpload returns a short-lived PUT URL for objectKey.
func (c *Client) SignUpload(ctx context.Context, objectKey string, expiresIn time.Duration) (*SignedUpload, error) {
	path := fmt.Sprintf("/storage/v1/object/upload/sign/%s/%s",
		url.PathEscape(c.bucket), escapeKey(objectKey))
	body := map[string]any{"expiresIn": int(expiresIn.Seconds())}
	var resp struct {
		URL   string `json:"url"`
		Token string `json:"token"`
	}
	if err := c.postJSON(ctx, path, body, &resp); err != nil {
		return nil, fmt.Errorf("sign upload: %w", err)
	}
	signed := resp.URL
	if signed == "" {
		return nil, fmt.Errorf("sign upload: empty url")
	}
	if !strings.HasPrefix(signed, "http") {
		signed = c.baseURL + "/storage/v1" + signed
	}
	headers := map[string]string{}
	if resp.Token != "" {
		headers["Authorization"] = "Bearer " + resp.Token
		// Supabase upload-sign often expects token as query or Authorization.
	}
	return &SignedUpload{URL: signed, Headers: headers, Token: resp.Token}, nil
}

// SignDownload returns a short-lived GET URL for objectKey.
func (c *Client) SignDownload(ctx context.Context, objectKey string, expiresIn time.Duration) (string, error) {
	path := fmt.Sprintf("/storage/v1/object/sign/%s/%s",
		url.PathEscape(c.bucket), escapeKey(objectKey))
	body := map[string]any{"expiresIn": int(expiresIn.Seconds())}
	var resp struct {
		SignedURL string `json:"signedURL"`
		SignedUrl string `json:"signedUrl"`
	}
	if err := c.postJSON(ctx, path, body, &resp); err != nil {
		return "", fmt.Errorf("sign download: %w", err)
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

type ObjectInfo struct {
	Exists      bool
	ContentType string
	Size        int64
}

// StatObject checks whether an object exists (HEAD). Size/ContentType when available.
func (c *Client) StatObject(ctx context.Context, objectKey string) (*ObjectInfo, error) {
	path := fmt.Sprintf("/storage/v1/object/%s/%s",
		url.PathEscape(c.bucket), escapeKey(objectKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("apikey", c.serviceKey)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)

	if res.StatusCode == http.StatusNotFound {
		return &ObjectInfo{Exists: false}, nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("stat object: status %d", res.StatusCode)
	}
	info := &ObjectInfo{
		Exists:      true,
		ContentType: res.Header.Get("Content-Type"),
	}
	if cl := res.Header.Get("Content-Length"); cl != "" {
		var n int64
		_, _ = fmt.Sscan(cl, &n)
		info.Size = n
	}
	return info, nil
}

// RemoveObject best-effort deletes an object. Soft-fail callers should ignore errors.
func (c *Client) RemoveObject(ctx context.Context, objectKey string) error {
	path := fmt.Sprintf("/storage/v1/object/%s/%s",
		url.PathEscape(c.bucket), escapeKey(objectKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("apikey", c.serviceKey)
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	if res.StatusCode == http.StatusNotFound {
		return nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("remove object: status %d", res.StatusCode)
	}
	return nil
}

func (c *Client) postJSON(ctx context.Context, path string, body any, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("apikey", c.serviceKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("status %d: %s", res.StatusCode, truncate(string(b), 200))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil
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
