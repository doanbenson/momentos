package ffmpeg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// #region agent log
func agentDbg(hypothesisID, location, message string, data map[string]any) {
	payload := map[string]any{
		"sessionId":    "ae9acc",
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
		"runId":        "post-fix",
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	// Prefer repo-root debug-ae9acc.log (workspace ingest path).
	candidates := []string{}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "..", "..", "..", "..", "debug-ae9acc.log"), // cwd=internal/ffmpeg
			filepath.Join(wd, "..", "..", "debug-ae9acc.log"),             // cwd=services/render-service
			filepath.Join(wd, "debug-ae9acc.log"),
		)
	}
	candidates = append(candidates,
		filepath.Join("..", "..", "..", "..", "debug-ae9acc.log"),
		filepath.Join("..", "..", "debug-ae9acc.log"),
		"debug-ae9acc.log",
	)
	written := false
	for _, p := range candidates {
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			continue
		}
		_, _ = f.Write(append(b, '\n'))
		_ = f.Close()
		written = true
		break
	}
	_ = written
}

// scaleSegments returns each scale=… filter argument for audit logs.
func scaleSegments(filter string) []string {
	out := make([]string, 0, 8)
	rest := filter
	for {
		i := strings.Index(rest, "scale=")
		if i < 0 {
			break
		}
		rest = rest[i:]
		end := strings.IndexAny(rest, ",;[")
		if end < 0 {
			out = append(out, rest)
			break
		}
		out = append(out, rest[:end])
		rest = rest[end:]
	}
	return out
}

// #endregion

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

	// #region agent log
	segs := scaleSegments(filter)
	frameVarScale := 0
	for _, s := range segs {
		if strings.Contains(s, "*t") || strings.Contains(s, "(t") ||
			strings.Contains(s, "*n") || strings.Contains(s, "(n") ||
			strings.Contains(s, "pos") {
			frameVarScale++
		}
	}
	agentDbg("H1", "runner.go:Render:pre-exec", "ffmpeg filter scale audit", map[string]any{
		"goos":              runtime.GOOS,
		"bin":               r.bin(),
		"scaleCount":        len(segs),
		"scaleSegments":     segs,
		"frameVarScaleCount": frameVarScale,
		"hasCropT":          strings.Contains(filter, "sin(2*PI*t"),
		"hasZoompan":        strings.Contains(filter, "zoompan="),
		"hasAct1Overscan":   strings.Contains(filter, fmt.Sprintf("scale=%d:%d", Act1OverscanWidth, Act1OverscanHeight)),
		"hasAct3Overscan":   strings.Contains(filter, fmt.Sprintf("scale=%d:%d", Act3DriftOverscanWidth, Act3DriftOverscanHeight)),
	})
	// #endregion

	args := make([]string, 0, 48)
	args = append(args, "-y", "-hide_banner", "-loglevel", "error")
	// Serialize filter_complex so N photo branches don't each allocate full frame pools.
	args = append(args, "-filter_threads", "1", "-filter_complex_threads", "1")
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
	// #region agent log
	joinedArgs := strings.Join(args, " ")
	agentDbg("G", "runner.go:Render:args", "ffmpeg invoke sizing", map[string]any{
		"pictureCount":   len(plan.PictureClips()),
		"clipCount":      len(plan.Clips),
		"filterLen":      len(filter),
		"fps18Count":     strings.Count(filter, "fps=18"),
		"loopFilterCount": strings.Count(filter, "loop=loop="),
		"hasInputLoop1":  strings.Contains(joinedArgs, "-loop 1"),
		"hasAcopy":       strings.Contains(filter, "acopy["),
		"hasFontfile":    strings.Contains(filter, "fontfile="),
		"filterThreads":  strings.Contains(joinedArgs, "-filter_threads 1"),
	})
	// #endregion

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
		// #region agent log
		agentDbg("H1", "runner.go:Render:ffmpeg-fail", "ffmpeg encode failed", map[string]any{
			"exit":             exit,
			"stderrHead":       truncate(msg, 512),
			"parsedScaleInit":  strings.Contains(msg, "Parsed_scale") && strings.Contains(msg, "init eval_mode"),
			"hasFontconfig":    strings.Contains(msg, "Fontconfig"),
			"frameVarScaleCount": frameVarScale,
			"goos":             runtime.GOOS,
		})
		// #endregion
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
	AssetID      string       `json:"assetId"`
	Order        int          `json:"order"`
	StartMs      int          `json:"startMs"`
	EndMs        int          `json:"endMs"`
	Role         ClipRole     `json:"role"`
	Act          ClipAct      `json:"act"`
	MotionPreset MotionPreset `json:"motionPreset,omitempty"`
	VisualStyle  VisualStyle  `json:"visualStyle"`
	TransitionIn TransitionIn `json:"transitionIn"`
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
			AssetID:      c.AssetID,
			Order:        c.Order,
			StartMs:      c.StartMs,
			EndMs:        c.EndMs,
			Type:         a.Type,
			LocalPath:    a.LocalPath,
			Role:         c.Role,
			Act:          c.Act,
			MotionPreset: c.MotionPreset,
			VisualStyle:  c.VisualStyle,
			TransitionIn: c.TransitionIn,
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

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}
