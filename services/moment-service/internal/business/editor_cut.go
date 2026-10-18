package business

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// clipEdit is one stored Cut clip. startMs/endMs are output placement.
// sourceInMs/sourceOutMs are the inclusive-exclusive file window (sourceIn defaults to 0).
type clipEdit struct {
	AssetID      uuid.UUID `json:"assetId"`
	Order        int       `json:"order"`
	StartMs      int       `json:"startMs"`
	EndMs        int       `json:"endMs"`
	SourceInMs   *int      `json:"sourceInMs,omitempty"`
	SourceOutMs  *int      `json:"sourceOutMs,omitempty"`
	Role         string    `json:"role,omitempty"`
	Act          *int      `json:"act,omitempty"`
	MotionPreset *string   `json:"motionPreset,omitempty"`
	VisualStyle  *string   `json:"visualStyle,omitempty"`
	TransitionIn *string   `json:"transitionIn,omitempty"`
}

// captionEdit matches Cut captions: {text, startMs, endMs} on the output timeline.
type captionEdit struct {
	Text    string `json:"text"`
	StartMs int    `json:"startMs"`
	EndMs   int    `json:"endMs"`
}

func clipEditRequested(p EditorPatch) bool {
	return p.Replace != nil || p.RemoveAssetID != nil || len(p.ReorderAssetIDs) > 0 || len(p.Trims) > 0
}

// applyCutEdits runs replace → remove → reorder → trims, then packs the timeline
// from 0 and shifts Reflection captions when that clip's sourceInMs changes.
func applyCutEdits(clipsRaw, captionsRaw json.RawMessage, patch EditorPatch) (clipsJSON, captionsJSON json.RawMessage, assetIDs []uuid.UUID, err error) {
	clips, err := decodeClips(clipsRaw)
	if err != nil {
		return nil, nil, nil, err
	}
	captions, err := decodeCaptions(captionsRaw)
	if err != nil {
		return nil, nil, nil, err
	}

	oldSourceIn := 0
	refStart, refEnd := 0, 0
	trackReflection := false
	if idx := reflectionIndex(clips); idx >= 0 {
		trackReflection = true
		oldSourceIn = sourceInOrZero(clips[idx])
		refStart = clips[idx].StartMs
		refEnd = clips[idx].EndMs
	}

	if patch.Replace != nil {
		if err := replaceClip(clips, patch.Replace.FromAssetID, patch.Replace.ToAssetID); err != nil {
			return nil, nil, nil, err
		}
	}
	if patch.RemoveAssetID != nil {
		clips, err = removeClip(clips, *patch.RemoveAssetID)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if len(patch.ReorderAssetIDs) > 0 {
		clips, err = reorderClips(clips, patch.ReorderAssetIDs)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if err := applyTrims(clips, patch.Trims); err != nil {
		return nil, nil, nil, err
	}

	reflowClips(clips)

	if trackReflection {
		if idx := reflectionIndex(clips); idx >= 0 {
			delta := oldSourceIn - sourceInOrZero(clips[idx])
			shiftReflectionCaptions(captions, refStart, refEnd, delta)
		}
	}

	clipsJSON, err = json.Marshal(clips)
	if err != nil {
		return nil, nil, nil, err
	}
	captionsJSON, err = json.Marshal(captions)
	if err != nil {
		return nil, nil, nil, err
	}
	assetIDs = make([]uuid.UUID, len(clips))
	for i, c := range clips {
		assetIDs[i] = c.AssetID
	}
	return clipsJSON, captionsJSON, assetIDs, nil
}

func decodeClips(raw json.RawMessage) ([]clipEdit, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return []clipEdit{}, nil
	}
	var clips []clipEdit
	if err := json.Unmarshal(trimmed, &clips); err != nil {
		return nil, fmt.Errorf("%w: clips: %v", ErrInvalid, err)
	}
	if clips == nil {
		clips = []clipEdit{}
	}
	return clips, nil
}

func decodeCaptions(raw json.RawMessage) ([]captionEdit, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return []captionEdit{}, nil
	}
	var captions []captionEdit
	if err := json.Unmarshal(trimmed, &captions); err != nil {
		return nil, fmt.Errorf("%w: captions: %v", ErrInvalid, err)
	}
	if captions == nil {
		captions = []captionEdit{}
	}
	return captions, nil
}

func reflectionIndex(clips []clipEdit) int {
	for i := range clips {
		if clips[i].Role == "reflection" {
			return i
		}
	}
	return -1
}

func sourceInOrZero(c clipEdit) int {
	if c.SourceInMs == nil {
		return 0
	}
	return *c.SourceInMs
}

// clipStoredDuration is the window kept across replace: sourceOut-sourceIn when
// both source fields are set, otherwise the previous timeline length.
func clipStoredDuration(c clipEdit) int {
	if c.SourceInMs != nil && c.SourceOutMs != nil {
		if d := *c.SourceOutMs - *c.SourceInMs; d > 0 {
			return d
		}
	}
	return timelineDuration(c)
}

func timelineDuration(c clipEdit) int {
	d := c.EndMs - c.StartMs
	if d < 0 {
		return 0
	}
	return d
}

// clipPlaybackDuration is the packed timeline length after edits.
func clipPlaybackDuration(c clipEdit) int {
	if c.SourceInMs != nil && c.SourceOutMs != nil && *c.SourceOutMs > *c.SourceInMs {
		return *c.SourceOutMs - *c.SourceInMs
	}
	return timelineDuration(c)
}

func replaceClip(clips []clipEdit, from, to uuid.UUID) error {
	for i := range clips {
		if clips[i].AssetID != from {
			continue
		}
		clips[i].AssetID = to
		dur := clipStoredDuration(clips[i])
		in := 0
		out := dur
		clips[i].SourceInMs = &in
		clips[i].SourceOutMs = &out
		return nil
	}
	return fmt.Errorf("%w: fromAssetId not in cut", ErrInvalid)
}

func removeClip(clips []clipEdit, id uuid.UUID) ([]clipEdit, error) {
	idx := -1
	for i := range clips {
		if clips[i].AssetID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("%w: removeAssetId not in cut", ErrInvalid)
	}
	if len(clips) <= 1 {
		return nil, fmt.Errorf("%w: cannot remove last asset", ErrInvalid)
	}
	out := make([]clipEdit, 0, len(clips)-1)
	out = append(out, clips[:idx]...)
	out = append(out, clips[idx+1:]...)
	return out, nil
}

func reorderClips(clips []clipEdit, ids []uuid.UUID) ([]clipEdit, error) {
	current := make([]uuid.UUID, len(clips))
	byID := make(map[uuid.UUID]clipEdit, len(clips))
	for i, c := range clips {
		current[i] = c.AssetID
		byID[c.AssetID] = c
	}
	if err := validateReorder(current, ids); err != nil {
		return nil, err
	}
	out := make([]clipEdit, len(ids))
	for i, id := range ids {
		out[i] = byID[id]
	}
	return out, nil
}

func applyTrims(clips []clipEdit, trims []ClipTrim) error {
	for _, trim := range trims {
		if trim.SourceOutMs <= trim.SourceInMs {
			return fmt.Errorf("%w: sourceOutMs must be greater than sourceInMs", ErrInvalid)
		}
		found := false
		for i := range clips {
			if clips[i].AssetID != trim.AssetID {
				continue
			}
			in := trim.SourceInMs
			out := trim.SourceOutMs
			clips[i].SourceInMs = &in
			clips[i].SourceOutMs = &out
			found = true
			break
		}
		if !found {
			return fmt.Errorf("%w: trim assetId not in cut", ErrInvalid)
		}
	}
	return nil
}

// reflowClips packs clips from timeline 0. Duration is the source window when
// both ends are set and sourceOut > sourceIn, otherwise the previous end-start.
func reflowClips(clips []clipEdit) {
	cursor := 0
	for i := range clips {
		dur := clipPlaybackDuration(clips[i])
		clips[i].Order = i
		clips[i].StartMs = cursor
		clips[i].EndMs = cursor + dur
		cursor += dur
	}
}

// shiftReflectionCaptions moves cues that overlap the Reflection clip's timeline
// window (captured before reflow) by oldSourceIn-newSourceIn.
// A missing window shifts every caption; those cues are the Reflection's.
func shiftReflectionCaptions(captions []captionEdit, refStart, refEnd, delta int) {
	if delta == 0 || len(captions) == 0 {
		return
	}
	bounded := refEnd > refStart
	for i := range captions {
		if bounded && !captionOverlaps(captions[i], refStart, refEnd) {
			continue
		}
		captions[i].StartMs += delta
		captions[i].EndMs += delta
	}
}

func captionOverlaps(c captionEdit, start, end int) bool {
	return c.StartMs < end && c.EndMs > start
}
