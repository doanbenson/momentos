package ffmpeg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Runner invokes local ffmpeg. Keep all encoding details here — not in ml/.
type Runner struct {
	// Bin is the ffmpeg binary. Empty means "ffmpeg" on PATH.
	Bin string
	// Timeout bounds a single encode. Zero uses default (10m).
	Timeout time.Duration
}

func (r *Runner) bin() string {
	if r.Bin != "" {
		return r.Bin
	}
	return "ffmpeg"
}

func (r *Runner) timeout() time.Duration {
	if r.Timeout > 0 {
		return r.Timeout
	}
	return 10 * time.Minute
}

// Render encodes plan to outPath as H.264/AAC MP4 (1080×1920@30).
// Does not write a placeholder file on failure.
func (r *Runner) Render(plan Plan, outPath string) error {
	if err := plan.Validate(); err != nil {
		return fmt.Errorf("plan: %w", err)
	}

	filter, vLabel, aLabel, err := buildFilterComplex(plan)
	if err != nil {
		return fmt.Errorf("filter_complex: %w", err)
	}

	args := make([]string, 0, 32)
	args = append(args, "-y", "-hide_banner", "-loglevel", "error")
	args = appendInputArgs(args, plan)
	args = append(args,
		"-filter_complex", filter,
		"-map", "["+vLabel+"]",
		"-map", "["+aLabel+"]",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-pix_fmt", "yuv420p",
		"-r", fmt.Sprintf("%d", FrameRate),
		"-c:a", "aac",
		"-b:a", "192k",
		"-movflags", "+faststart",
		"-shortest",
		outPath,
	)

	ctx, cancel := context.WithTimeout(context.Background(), r.timeout())
	defer cancel()

	cmd := exec.CommandContext(ctx, r.bin(), args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Remove partial output so callers never treat a stub/partial as success.
		_ = os.Remove(outPath)
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("ffmpeg timed out after %s", r.timeout())
		}
		exit := -1
		if ee, ok := err.(*exec.ExitError); ok {
			exit = ee.ExitCode()
		}
		msg := stderr.String()
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("ffmpeg exit %d: %s", exit, truncate(msg, 2<<10))
	}

	fi, err := os.Stat(outPath)
	if err != nil || fi.Size() == 0 {
		_ = os.Remove(outPath)
		return fmt.Errorf("ffmpeg produced empty or missing output")
	}
	return nil
}

// CutWire is the Cut JSON shape plus local path map for RenderFromCut.
type CutWire struct {
	ID             string                `json:"id"`
	MomentID       string                `json:"momentId"`
	Version        int                   `json:"version"`
	SongPickID     string                `json:"songPickId"`
	StyleProfileID string                `json:"styleProfileId"`
	Clips          []CutWireClip         `json:"clips"`
	Captions       []Caption             `json:"captions"`
	Markers        []Marker              `json:"markers"`
	Assets         map[string]AssetLocal `json:"assets"` // assetId → local resolve
	EditAudio      string                `json:"editAudioPath"`
}

type CutWireClip struct {
	AssetID string   `json:"assetId"`
	Order   int      `json:"order"`
	StartMs int      `json:"startMs"`
	EndMs   int      `json:"endMs"`
	Role    ClipRole `json:"role"`
}

type AssetLocal struct {
	Type      AssetType `json:"type"`
	LocalPath string    `json:"localPath"`
}

// RenderFromCut is the seam from SPEC: cutJSON (with resolved local paths) → MP4.
func (r *Runner) RenderFromCut(cutJSON []byte, outPath string) error {
	var wire CutWire
	if err := json.Unmarshal(cutJSON, &wire); err != nil {
		return fmt.Errorf("cut JSON: %w", err)
	}
	plan, err := planFromWire(wire)
	if err != nil {
		return err
	}
	return r.Render(plan, outPath)
}

func planFromWire(w CutWire) (Plan, error) {
	p := Plan{
		MomentID:       w.MomentID,
		CutID:          w.ID,
		CutVersion:     w.Version,
		EditAudioPath:  w.EditAudio,
		StyleProfileID: w.StyleProfileID,
		Clips:          make([]ClipInput, 0, len(w.Clips)),
		Captions:       append([]Caption(nil), w.Captions...),
		Markers:        append([]Marker(nil), w.Markers...),
	}
	for _, c := range w.Clips {
		a, ok := w.Assets[c.AssetID]
		if !ok {
			return Plan{}, fmt.Errorf("asset %s: missing type/path in cut JSON assets map", c.AssetID)
		}
		p.Clips = append(p.Clips, ClipInput{
			AssetID:   c.AssetID,
			Order:     c.Order,
			StartMs:   c.StartMs,
			EndMs:     c.EndMs,
			Type:      a.Type,
			LocalPath: a.LocalPath,
			Role:      c.Role,
		})
	}
	return p, nil
}

// Available reports whether the ffmpeg binary can be executed.
func (r *Runner) Available() bool {
	cmd := exec.Command(r.bin(), "-version")
	return cmd.Run() == nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
