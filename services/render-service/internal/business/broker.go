package business

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/doanbenson/momentos/services/render-service/internal/database"
)

// MomentBroker claims/completes render jobs on moment-service.
type MomentBroker struct {
	BaseURL       string
	InternalToken string
	WorkerID      string
	HTTP          *http.Client
}

func NewMomentBrokerFromEnv() *MomentBroker {
	base := strings.TrimRight(os.Getenv("MOMENT_SERVICE_URL"), "/")
	if base == "" {
		return nil
	}
	wid := os.Getenv("WORKER_ID")
	if wid == "" {
		wid = "render-" + hostname()
	}
	return &MomentBroker{
		BaseURL:       base,
		InternalToken: os.Getenv("INTERNAL_SERVICE_TOKEN"),
		WorkerID:      wid,
		HTTP:          &http.Client{Timeout: 60 * time.Second},
	}
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

type claimEnvelope struct {
	ID       string          `json:"id"`
	MomentID string          `json:"momentId"`
	JobType  string          `json:"jobType"`
	Payload  json.RawMessage `json:"payload"`
}

type renderPayload struct {
	JobType            string `json:"jobType"`
	JobID              string `json:"jobId"`
	MomentID           string `json:"momentId"`
	CutVersion         int    `json:"cutVersion"`
	SongPickID         string `json:"songPickId"`
	EditAudioURL       string `json:"editAudioUrl"`
	ReflectionQuestion string `json:"reflectionQuestion"`
	Cut                struct {
		ID         string          `json:"id"`
		MomentID   string          `json:"momentId"`
		Version    int             `json:"version"`
		SongPickID string          `json:"songPickId"`
		Clips      json.RawMessage `json:"clips"`
	} `json:"cut"`
	Assets []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		StorageURL string `json:"storageUrl"`
		DurationMs int    `json:"durationMs"`
	} `json:"assets"`
}

// PollClaim pulls one render job from moment-service into the local store.
func (b *MomentBroker) PollClaim(ctx context.Context, store *database.Store) error {
	if b == nil || b.BaseURL == "" {
		return nil
	}
	body, _ := json.Marshal(map[string]string{
		"jobType":  "render",
		"workerId": b.WorkerID,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.BaseURL+"/v1/internal/jobs/claim", bytes.NewReader(body))
	if err != nil {
		return err
	}
	b.setAuth(req)
	req.Header.Set("Content-Type", "application/json")
	res, err := b.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode == http.StatusNoContent {
		return nil
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("claim %d: %s", res.StatusCode, truncate(raw))
	}
	var env claimEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	if len(bytes.TrimSpace(env.Payload)) == 0 || string(env.Payload) == "null" {
		log.Printf("render-service: broker claim job=%s has empty payload; skipping", env.ID)
		return nil
	}
	var payload renderPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("payload: %w", err)
	}
	jobID := env.ID
	if jobID == "" {
		jobID = payload.JobID
	}
	cutJSON, _ := json.Marshal(map[string]any{"clips": json.RawMessage(payload.Cut.Clips)})
	if len(payload.Cut.Clips) == 0 {
		cutJSON = []byte(`{"clips":[]}`)
	} else {
		cutJSON, _ = json.Marshal(map[string]json.RawMessage{"clips": payload.Cut.Clips})
	}
	assets := make([]database.AssetRef, 0, len(payload.Assets))
	for _, a := range payload.Assets {
		assets = append(assets, database.AssetRef{ID: a.ID, Type: a.Type, URL: a.StorageURL})
	}
	cutID := payload.Cut.ID
	cutVer := payload.CutVersion
	if cutVer == 0 {
		cutVer = payload.Cut.Version
	}
	job := &database.Job{
		ID:                 jobID,
		MomentID:           env.MomentID,
		CutID:              cutID,
		CutVersion:         cutVer,
		SongPickID:         payload.SongPickID,
		EditAudioURL:       payload.EditAudioURL,
		ReflectionQuestion: payload.ReflectionQuestion,
		CutJSON:            cutJSON,
		Assets:             assets,
		Status:             database.StatusQueued,
		MaxAttempts:        3,
	}
	if job.MomentID == "" {
		job.MomentID = payload.MomentID
	}
	if err := store.Create(job); err != nil {
		// Already claimed locally — ignore duplicate
		log.Printf("render-service: broker enqueue job=%s: %v", jobID, err)
	}
	return nil
}

// CompleteSuccess notifies moment-service.
func (b *MomentBroker) CompleteSuccess(ctx context.Context, job *database.Job) error {
	if b == nil || b.BaseURL == "" || job == nil {
		return nil
	}
	body, _ := json.Marshal(map[string]any{
		"ok":               true,
		"renderStorageKey": job.OutputKey,
		"renderUrl":        job.RenderURL,
	})
	return b.complete(ctx, job.ID, body)
}

// CompleteFailure notifies moment-service of encode failure.
func (b *MomentBroker) CompleteFailure(ctx context.Context, jobID, code, message string) error {
	if b == nil || b.BaseURL == "" {
		return nil
	}
	body, _ := json.Marshal(map[string]any{
		"ok":           false,
		"errorCode":    code,
		"errorMessage": message,
	})
	return b.complete(ctx, jobID, body)
}

func (b *MomentBroker) complete(ctx context.Context, jobID string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.BaseURL+"/v1/internal/jobs/"+jobID+"/complete", bytes.NewReader(body))
	if err != nil {
		return err
	}
	b.setAuth(req)
	req.Header.Set("Content-Type", "application/json")
	res, err := b.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return fmt.Errorf("complete %d: %s", res.StatusCode, truncate(raw))
	}
	return nil
}

func (b *MomentBroker) setAuth(req *http.Request) {
	if b.InternalToken != "" {
		req.Header.Set("Authorization", "Bearer "+b.InternalToken)
		req.Header.Set("X-Momentos-Internal-Token", b.InternalToken)
	}
}

func truncate(b []byte) string {
	s := string(b)
	if len(s) > 200 {
		return s[:200]
	}
	return s
}
