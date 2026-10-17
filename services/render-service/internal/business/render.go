package business

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/doanbenson/momentos/services/render-service/internal/database"
	"github.com/doanbenson/momentos/services/render-service/internal/ffmpeg"
)

const defaultSongPickID = "live-a-life-you-will-remember"

// Service turns a Cut into a Render artifact via ffmpeg.
type Service struct {
	Store  *database.Store
	Runner *ffmpeg.Runner
	// EditAudioPath is the MVP local fallback when job has no editAudioUrl.
	EditAudioPath string
	// StorageUpload is optional; nil skips upload and leaves local path only.
	StorageUpload func(localPath, objectKey string) (renderURL string, err error)
	// NotifyMoment is optional callback to moment-service (ready | failed).
	NotifyMoment func(job *database.Job) error
	HTTPClient   *http.Client
}

func (s *Service) http() *http.Client {
	if s.HTTPClient != nil {
		return s.HTTPClient
	}
	return &http.Client{Timeout: 2 * time.Minute}
}

// Enqueue validates minimum fields and stores a queued job.
func (s *Service) Enqueue(job *database.Job) (*database.Job, error) {
	if job.ID == "" {
		return nil, Terminal("job id required")
	}
	if job.MomentID == "" || job.CutID == "" {
		return nil, Terminal("momentId and cutId required")
	}
	if job.SongPickID == "" {
		job.SongPickID = defaultSongPickID
	}
	job.Status = database.StatusQueued
	job.MaxAttempts = 3 // ≤2 retries after first attempt
	if err := s.Store.Create(job); err != nil {
		return nil, err
	}
	created, ok := s.Store.Get(job.ID)
	if !ok {
		return nil, fmt.Errorf("job %s missing after create", job.ID)
	}
	return created, nil
}

// ProcessJob runs one attempt: scratch → download → encode → upload → notify.
// Returns terminal vs retryable via IsTerminal.
func (s *Service) ProcessJob(job *database.Job) error {
	if _, err := s.Store.MarkRunning(job.ID); err != nil {
		return err
	}
	fresh, _ := s.Store.Get(job.ID)
	if fresh != nil {
		job = fresh
	}

	scratch, err := os.MkdirTemp("", "render-"+job.ID+"-*")
	if err != nil {
		return fmt.Errorf("scratch dir: %w", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(scratch); rmErr != nil {
			log.Printf("render-service: cleanup scratch %s: %v", scratch, rmErr)
		}
	}()

	plan, err := s.buildPlan(job, scratch)
	if err != nil {
		return err
	}

	outPath := filepath.Join(scratch, "out.mp4")
	start := time.Now()
	if err := s.Runner.Render(plan, outPath); err != nil {
		log.Printf("render-service: job=%s ffmpeg failed after %s: %v",
			job.ID, time.Since(start).Round(time.Millisecond), err)
		return fmt.Errorf("encode: %w", err)
	}
	log.Printf("render-service: job=%s moment=%s cut=%s v=%d encoded in %s",
		job.ID, job.MomentID, job.CutID, job.CutVersion, time.Since(start).Round(time.Millisecond))

	objectKey := fmt.Sprintf("moments/%s/renders/%d.mp4", job.MomentID, job.CutVersion)
	renderURL := "file://" + outPath
	if s.StorageUpload != nil {
		url, upErr := s.StorageUpload(outPath, objectKey)
		if upErr != nil {
			return fmt.Errorf("upload: %w", upErr)
		}
		renderURL = url
	} else {
		// Persist a copy outside scratch so cleanup does not delete the only artifact.
		persist := filepath.Join(os.TempDir(), fmt.Sprintf("momentos-render-%s-v%d.mp4", job.MomentID, job.CutVersion))
		if cpErr := copyFile(outPath, persist); cpErr != nil {
			return fmt.Errorf("persist local render: %w", cpErr)
		}
		renderURL = "file://" + persist
		log.Printf("render-service: StorageUpload unset; kept local copy %s", persist)
	}

	updated, err := s.Store.MarkSucceeded(job.ID, objectKey, renderURL)
	if err != nil {
		return err
	}
	if s.NotifyMoment != nil {
		if nErr := s.NotifyMoment(updated); nErr != nil {
			log.Printf("render-service: notify moment-service job=%s: %v", job.ID, nErr)
		}
	}
	return nil
}

func (s *Service) buildPlan(job *database.Job, scratch string) (ffmpeg.Plan, error) {
	if len(job.Assets) == 0 {
		return ffmpeg.Plan{}, Terminal("job missing assets (type + url map)")
	}

	editPath, err := s.resolveEditAudio(job, scratch)
	if err != nil {
		return ffmpeg.Plan{}, err
	}

	byID := make(map[string]database.AssetRef, len(job.Assets))
	for _, a := range job.Assets {
		byID[a.ID] = a
	}

	// Prefer CutJSON clips when present; else require assets alone is insufficient.
	var clips []ffmpeg.ClipInput
	if len(job.CutJSON) > 0 {
		wireClips, wErr := parseCutClips(job.CutJSON)
		if wErr != nil {
			return ffmpeg.Plan{}, Terminal("cut JSON: %v", wErr)
		}
		for _, c := range wireClips {
			a, ok := byID[c.AssetID]
			if !ok {
				return ffmpeg.Plan{}, Terminal("asset %s: not on job assets list", c.AssetID)
			}
			local, dErr := s.downloadAsset(a, scratch)
			if dErr != nil {
				return ffmpeg.Plan{}, fmt.Errorf("download asset %s: %w", a.ID, dErr)
			}
			clips = append(clips, ffmpeg.ClipInput{
				AssetID:   c.AssetID,
				Order:     c.Order,
				StartMs:   c.StartMs,
				EndMs:     c.EndMs,
				Type:      ffmpeg.AssetType(a.Type),
				LocalPath: local,
			})
		}
	} else {
		return ffmpeg.Plan{}, Terminal("job missing Cut JSON clips")
	}

	plan := ffmpeg.Plan{
		MomentID:      job.MomentID,
		CutID:         job.CutID,
		CutVersion:    job.CutVersion,
		EditAudioPath: editPath,
		Clips:         clips,
	}
	if err := plan.Validate(); err != nil {
		return ffmpeg.Plan{}, Terminal("%s", err.Error())
	}
	return plan, nil
}

func (s *Service) resolveEditAudio(job *database.Job, scratch string) (string, error) {
	if job.EditAudioURL != "" {
		dest := filepath.Join(scratch, "edit-audio")
		if err := downloadURL(s.http(), job.EditAudioURL, dest); err != nil {
			return "", fmt.Errorf("download Edit Audio: %w", err)
		}
		return dest, nil
	}
	if s.EditAudioPath != "" {
		if _, err := os.Stat(s.EditAudioPath); err != nil {
			return "", Terminal("Edit Audio config path %s: %v", s.EditAudioPath, err)
		}
		return s.EditAudioPath, nil
	}
	return "", Terminal("missing Edit Audio (job editAudioUrl or RENDER_EDIT_AUDIO_PATH)")
}

func (s *Service) downloadAsset(a database.AssetRef, scratch string) (string, error) {
	if a.Type == "" {
		return "", Terminal("asset %s: missing type", a.ID)
	}
	if a.URL == "" {
		return "", Terminal("asset %s: missing url", a.ID)
	}
	// Local file:// or plain path for hackathon/dev without signed URLs.
	if path, ok := localFileRef(a.URL); ok {
		if _, err := os.Stat(path); err != nil {
			return "", Terminal("asset %s local path: %v", a.ID, err)
		}
		return path, nil
	}
	dest := filepath.Join(scratch, "asset-"+a.ID)
	if err := downloadURL(s.http(), a.URL, dest); err != nil {
		return "", err
	}
	return dest, nil
}

type cutClip struct {
	AssetID string `json:"assetId"`
	Order   int    `json:"order"`
	StartMs int    `json:"startMs"`
	EndMs   int    `json:"endMs"`
}

func parseCutClips(cutJSON []byte) ([]cutClip, error) {
	var wrap struct {
		Clips []cutClip `json:"clips"`
	}
	if err := json.Unmarshal(cutJSON, &wrap); err != nil {
		return nil, err
	}
	if len(wrap.Clips) == 0 {
		return nil, fmt.Errorf("clips empty")
	}
	return wrap.Clips, nil
}

func downloadURL(client *http.Client, rawURL, dest string) error {
	resp, err := client.Get(rawURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, rawURL)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func localFileRef(u string) (string, bool) {
	if strings.HasPrefix(u, "file:") {
		parsed, err := url.Parse(u)
		if err != nil {
			return "", false
		}
		p := parsed.Path
		// Windows file:///C:/path → /C:/path
		if strings.HasPrefix(p, "/") && len(p) >= 3 && p[2] == ':' {
			p = p[1:]
		}
		if parsed.Host != "" && parsed.Host != "localhost" {
			p = `\\` + parsed.Host + strings.ReplaceAll(p, "/", `\`)
		}
		return p, true
	}
	// Absolute path without scheme — hackathon convenience.
	if filepath.IsAbs(u) {
		return u, true
	}
	return "", false
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// FinishAttempt applies retry policy after ProcessJob returns an error.
func (s *Service) FinishAttempt(jobID string, err error) {
	if err == nil {
		return
	}
	job, ok := s.Store.Get(jobID)
	if !ok {
		return
	}
	reason := err.Error()
	if IsTerminal(err) {
		failed, _ := s.Store.MarkFailed(jobID, reason, true)
		if s.NotifyMoment != nil && failed != nil {
			_ = s.NotifyMoment(failed)
		}
		log.Printf("render-service: job=%s terminal fail: %s", jobID, reason)
		return
	}
	if job.Attempts >= job.MaxAttempts {
		failed, _ := s.Store.MarkFailed(jobID, reason, false)
		if s.NotifyMoment != nil && failed != nil {
			_ = s.NotifyMoment(failed)
		}
		log.Printf("render-service: job=%s exhausted retries (%d): %s", jobID, job.Attempts, reason)
		return
	}
	_, _ = s.Store.Requeue(jobID, reason)
	log.Printf("render-service: job=%s retry scheduled (attempt %d/%d): %s",
		jobID, job.Attempts, job.MaxAttempts, reason)
}
