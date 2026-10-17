package business

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPError struct {
	Status int
	Body   []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("upstream %d: %s", e.Status, truncate(e.Body))
}

type PeerClient struct {
	BaseURL       string
	InternalToken string
	HTTP          *http.Client
}

func NewPeer(baseURL, token string) *PeerClient {
	return &PeerClient{
		BaseURL:       strings.TrimRight(baseURL, "/"),
		InternalToken: token,
		HTTP:          &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *PeerClient) Do(ctx context.Context, method, path, principal string, body []byte, extra http.Header) (int, []byte, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, rdr)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+c.InternalToken)
	req.Header.Set("X-Momentos-Internal-Token", c.InternalToken)
	if principal != "" {
		req.Header.Set("X-Momentos-Principal", principal)
	}
	for k, vals := range extra {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, err
	}
	return res.StatusCode, b, nil
}

func (c *PeerClient) JSON(ctx context.Context, method, path, principal string, in any, out any) error {
	var body []byte
	var err error
	if in != nil {
		body, err = json.Marshal(in)
		if err != nil {
			return err
		}
	}
	status, raw, err := c.Do(ctx, method, path, principal, body, nil)
	if err != nil {
		return err
	}
	if status >= 300 {
		return &HTTPError{Status: status, Body: raw}
	}
	if out == nil || status == http.StatusNoContent || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func truncate(b []byte) string {
	s := string(b)
	if len(s) > 240 {
		return s[:240]
	}
	return s
}
