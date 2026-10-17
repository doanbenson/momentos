package ffmpeg

import (
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

const (
	CanvasWidth  = 1080
	CanvasHeight = 1920
	FrameRate    = 30
)

// ClipInput is one Cut clip resolved to a local file for encode.
type ClipInput struct {
	AssetID   string
	Order     int
	StartMs   int
	EndMs     int
	Type      AssetType
	LocalPath string
}

// Plan is the typed encode input. Built by business after downloads.
type Plan struct {
	MomentID      string
	CutID         string
	CutVersion    int
	EditAudioPath string
	Clips         []ClipInput
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
	return nil
}
