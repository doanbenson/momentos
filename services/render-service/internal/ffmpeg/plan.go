package ffmpeg

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/doanbenson/momentos/services/render-service/internal/songpick"
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

// ClipAct mirrors OpenAPI CutClipAct.
type ClipAct int

const (
	ActReflection ClipAct = 1
	ActMontage    ClipAct = 2
	ActHero       ClipAct = 3
)

// MotionPreset mirrors OpenAPI CutMotionPreset.
type MotionPreset string

const (
	MotionHookSlowPush         MotionPreset = "hook_slow_push"
	MotionWhipPunchIn          MotionPreset = "whip_punch_in"
	MotionHandheldSuper8Drift  MotionPreset = "handheld_super8_drift"
	MotionNativeActionCrop     MotionPreset = "native_action_crop"
)

// VisualStyle mirrors OpenAPI CutVisualStyle.
type VisualStyle string

const (
	VisualClean916   VisualStyle = "clean_916"
	VisualSuper8Gate VisualStyle = "super8_gate"
)

// TransitionIn mirrors OpenAPI CutTransitionIn.
type TransitionIn string

const (
	TransitionNone       TransitionIn = "none"
	TransitionHardCut    TransitionIn = "hard_cut"
	TransitionWhiteFlash TransitionIn = "white_flash"
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
	// Super 8 / vintage print gate inset on the 1080×1920 canvas (life-on-film-v1).
	// 960×1280 is classic 3:4 vertical print ratio (@lifeonfilm.27).
	Super8GateWidth  = 960
	Super8GateHeight = 1280
	// Act1Overscan is static scale headroom before crop to canvas (Ken Burns / hold).
	// Never animate these with t/n/pos — scale init eval rejects frame vars.
	Act1OverscanWidth  = 1140
	Act1OverscanHeight = 2026
	// Act3DriftOverscan is static scale headroom before fixed 960×1280 crop weave.
	Act3DriftOverscanWidth  = 1000
	Act3DriftOverscanHeight = 1334
	// Act3CanvasPadColor is the warm-black letterbox behind the film print.
	Act3CanvasPadColor = "0x0C0B0A"
	// FlashFrames is white-flash duration (ADR 0021).
	FlashFrames = 3
	// JCutEditAudioVolume is Edit Audio level during J-cut pre-roll (ADR 0019).
	JCutEditAudioVolume = 0.18
	// StyleProfileLifeOnFilmV1 is the MVP Style Profile id (ADR 0013).
	// Mirrors songpick.StyleProfileID / packages/contracts/mvp-constants.json.
	StyleProfileLifeOnFilmV1 = "life-on-film-v1"
	// MsPerFrame at Canvas FrameRate (30fps → ~33.333ms); used for ±1 frame chorus-drop snap.
	MsPerFrame = 1000.0 / float64(FrameRate)
	// Clean916SlateColor is the voice-only Reflection slate (Film Grain dark).
	Clean916SlateColor = "0x12100E"
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
	AssetID      string
	Order        int
	StartMs      int
	EndMs        int
	Type         AssetType
	LocalPath    string
	Role         ClipRole
	Act          ClipAct      `json:"act"`
	MotionPreset MotionPreset `json:"motionPreset,omitempty"`
	VisualStyle  VisualStyle  `json:"visualStyle"`
	TransitionIn TransitionIn `json:"transitionIn"`
}

// Plan is the typed encode input. Built by business after downloads.
type Plan struct {
	MomentID       string
	CutID          string
	CutVersion     int
	EditAudioPath  string
	StyleProfileID string
	// OverlayPath optional Style Profile film-gate / burn PNG (system media).
	OverlayPath string
	// ReflectionQuestion is Act 1 prompt banner text from the render job (ADR 0020); not on Cut clips.
	ReflectionQuestion string
	Clips              []ClipInput
	Captions           []Caption
	Markers            []Marker
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
		if c.LocalPath == "" && !c.isClean916Slate() {
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
		if err := p.validateHeroOutroSpan(); err != nil {
			return err
		}
	}
	return nil
}

func (p Plan) validateHeroOutroSpan() error {
	var hero *ClipInput
	for i := range p.Clips {
		if p.Clips[i].Role == RoleHeroOutro {
			hero = &p.Clips[i]
		}
	}
	if hero == nil {
		return fmt.Errorf("Style Profile %s requires a clip with role hero_outro", StyleProfileLifeOnFilmV1)
	}
	expected := songpick.HeroOutroSpanBeats * 60_000 / songpick.NominalBPM
	tolerance := 60_000 / (2 * songpick.NominalBPM) // ±½ beat
	span := hero.DurationMs()
	delta := span - expected
	if delta < 0 {
		delta = -delta
	}
	if delta > tolerance {
		return fmt.Errorf(
			"hero_outro must be a legal %d-beat span (~%dms ±%dms at %d BPM); got %dms",
			songpick.HeroOutroSpanBeats, expected, tolerance, songpick.NominalBPM, span,
		)
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

// isClean916Slate is voice-only Reflection picture: clean_916 with no file (ADR 0020).
func (c ClipInput) isClean916Slate() bool {
	return c.Role == RoleReflection && c.VisualStyle == VisualClean916 &&
		c.Type == AssetPhoto && c.LocalPath == "" && c.MotionPreset == ""
}

// usesAct3Look reports whether Style Profile Act 3 look applies (all act=3; hero_outro fallback).
func (c ClipInput) usesAct3Look() bool {
	if c.Act == ActHero {
		return true
	}
	return c.Act == 0 && c.Role == RoleHeroOutro
}

// Act1PictureEndMs is the timeline end of Act 1 picture (Reflection / act=1 clips).
func (p Plan) Act1PictureEndMs() int {
	end := 0
	timeline := 0
	found := false
	for _, c := range p.PictureClips() {
		dur := c.DurationMs()
		isAct1 := c.Act == ActReflection || (c.Act == 0 && c.Role == RoleReflection)
		if isAct1 {
			end = timeline + dur
			found = true
		} else if found {
			break
		}
		timeline += dur
	}
	if found {
		return end
	}
	return 0
}

// JCutStartMs returns when Edit Audio J-cut pre-roll begins (marker or Act1End − pre-roll).
func (p Plan) JCutStartMs() int {
	act1End := p.Act1PictureEndMs()
	for _, m := range p.Markers {
		if m.Type == MarkerJCut {
			return SnapMsToFrame(m.AtMs)
		}
	}
	if act1End <= 0 {
		return 0
	}
	preRollMs := int(songpick.JCutPreRollSec * 1000)
	start := act1End - preRollMs
	if start < 0 {
		start = 0
	}
	return SnapMsToFrame(start)
}
