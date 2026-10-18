package ffmpeg

import (
	"encoding/json"
	"fmt"
	"sort"
)

// AssetType mirrors OpenAPI AssetType for encode branching.
type AssetType string

const (
	AssetPhoto AssetType = "photo"
	AssetVideo AssetType = "video"
	AssetVoice AssetType = "voice"
)

// ClipRole mirrors OpenAPI CutClipRole.
type ClipRole string

const (
	RoleReflection ClipRole = "reflection"
	RoleMontage    ClipRole = "montage"
	RoleHeroOutro  ClipRole = "hero_outro"
)

// MarkerType mirrors OpenAPI CutMarkerType.
type MarkerType string

const (
	MarkerJCut       MarkerType = "j_cut"
	MarkerChorusDrop MarkerType = "chorus_drop"
	MarkerFlash      MarkerType = "flash"
	MarkerSFX        MarkerType = "sfx"
)

const (
	CanvasWidth  = 1080
	CanvasHeight = 1920
	FrameRate    = 30
	// Act3PosterizeFPS is Style Profile life-on-film-v1 hold rate (no timeline stretch).
	Act3PosterizeFPS = 18
	// StyleProfileLifeOnFilmV1 is the MVP Style Profile id (ADR 0013).
	// Mirrors songpick.StyleProfileID / packages/contracts/mvp-constants.json.
	StyleProfileLifeOnFilmV1 = "life-on-film-v1"
	// MsPerFrame at Canvas FrameRate (30fps → ~33.333ms); used for ±1 frame chorus-drop snap.
	MsPerFrame = 1000.0 / float64(FrameRate)
)

// Caption is one Cut caption cue burned in Act 1 (no Whisper in render).
type Caption struct {
	Text    string `json:"text"`
	StartMs int    `json:"startMs"`
	EndMs   int    `json:"endMs"`
}

// Marker is one Cut marker executed in Act 2.
type Marker struct {
	Type MarkerType `json:"type"`
	AtMs int        `json:"atMs"`
	// SystemMediaID optional payload ref (workstream E); LocalPath when resolved.
	SystemMediaID string `json:"systemMediaId,omitempty"`
	LocalPath     string `json:"localPath,omitempty"`
}

// UnmarshalJSON accepts OpenAPI payload.systemMediaId as well as a top-level field.
func (m *Marker) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type          MarkerType             `json:"type"`
		AtMs          int                    `json:"atMs"`
		SystemMediaID string                 `json:"systemMediaId"`
		LocalPath     string                 `json:"localPath"`
		Payload       map[string]interface{} `json:"payload"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.Type = raw.Type
	m.AtMs = raw.AtMs
	m.SystemMediaID = raw.SystemMediaID
	m.LocalPath = raw.LocalPath
	if m.SystemMediaID == "" && raw.Payload != nil {
		if id, ok := raw.Payload["systemMediaId"].(string); ok {
			m.SystemMediaID = id
		}
	}
	return nil
}

// ClipInput is one Cut clip resolved to a local file for encode.
type ClipInput struct {
	AssetID   string
	Order     int
	StartMs   int
	EndMs     int
	Type      AssetType
	LocalPath string
	Role      ClipRole
}

// Plan is the typed encode input. Built by business after downloads.
type Plan struct {
	MomentID       string
	CutID          string
	CutVersion     int
	EditAudioPath  string
	StyleProfileID string
	Clips          []ClipInput
	Captions       []Caption
	Markers        []Marker
}

// DurationMs returns endMs - startMs.
func (c ClipInput) DurationMs() int {
	return c.EndMs - c.StartMs
}

// PictureClips returns photo/video clips sorted by order.
func (p Plan) PictureClips() []ClipInput {
	var out []ClipInput
	for _, c := range p.Clips {
		if c.Type == AssetPhoto || c.Type == AssetVideo {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Order < out[j].Order
	})
	return out
}

// VoiceClips returns voice clips sorted by order.
func (p Plan) VoiceClips() []ClipInput {
	var out []ClipInput
	for _, c := range p.Clips {
		if c.Type == AssetVoice {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Order < out[j].Order
	})
	return out
}

// PictureDurationMs is the concatenated picture timeline length.
func (p Plan) PictureDurationMs() int {
	total := 0
	for _, c := range p.PictureClips() {
		total += c.DurationMs()
	}
	return total
}

// Validate checks encode-required fields. Failures are terminal (no retry).
func (p Plan) Validate() error {
	if p.EditAudioPath == "" {
		return fmt.Errorf("missing Edit Audio path")
	}
	if len(p.Clips) == 0 {
		return fmt.Errorf("clips empty")
	}

	orders := make(map[int]string, len(p.Clips))
	for i, c := range p.Clips {
		if c.AssetID == "" {
			return fmt.Errorf("clip[%d]: missing assetId", i)
		}
		if c.LocalPath == "" {
			return fmt.Errorf("clip[%d] asset %s: missing local path", i, c.AssetID)
		}
		switch c.Type {
		case AssetPhoto, AssetVideo, AssetVoice:
		default:
			return fmt.Errorf("clip[%d] asset %s: missing or unknown type %q", i, c.AssetID, c.Type)
		}
		if c.EndMs <= c.StartMs {
			return fmt.Errorf("clip[%d] asset %s: endMs (%d) <= startMs (%d)", i, c.AssetID, c.EndMs, c.StartMs)
		}
		if prev, ok := orders[c.Order]; ok {
			return fmt.Errorf("duplicate clip order %d (assets %s and %s)", c.Order, prev, c.AssetID)
		}
		orders[c.Order] = c.AssetID
	}

	if len(p.PictureClips()) == 0 {
		return fmt.Errorf("no picture clips (photo/video); cannot encode video")
	}

	if err := p.validateCaptions(); err != nil {
		return err
	}
	if err := p.validateMarkers(); err != nil {
		return err
	}
	if p.StyleProfileID == StyleProfileLifeOnFilmV1 {
		if !p.hasRole(RoleHeroOutro) {
			return fmt.Errorf("Style Profile %s requires a clip with role hero_outro", StyleProfileLifeOnFilmV1)
		}
	}
	return nil
}

func (p Plan) hasRole(role ClipRole) bool {
	for _, c := range p.Clips {
		if c.Role == role {
			return true
		}
	}
	return false
}

func (p Plan) validateCaptions() error {
	for i, c := range p.Captions {
		if c.Text == "" {
			return fmt.Errorf("caption[%d]: empty text", i)
		}
		if c.EndMs <= c.StartMs {
			return fmt.Errorf("caption[%d]: endMs (%d) <= startMs (%d)", i, c.EndMs, c.StartMs)
		}
	}
	return nil
}

func (p Plan) validateMarkers() error {
	for i, m := range p.Markers {
		switch m.Type {
		case MarkerJCut, MarkerChorusDrop, MarkerFlash, MarkerSFX:
		default:
			return fmt.Errorf("marker[%d]: unknown type %q", i, m.Type)
		}
		if m.AtMs < 0 {
			return fmt.Errorf("marker[%d]: atMs (%d) < 0", i, m.AtMs)
		}
	}
	return nil
}

// SnapMsToFrame snaps timeline ms to the nearest canvas frame (chorus-drop ±1 frame target).
func SnapMsToFrame(ms int) int {
	if ms < 0 {
		return 0
	}
	frame := int(float64(ms)/MsPerFrame + 0.5)
	return int(float64(frame)*MsPerFrame + 0.5)
}
