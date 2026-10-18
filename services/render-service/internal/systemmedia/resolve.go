// Package systemmedia resolves Style Profile SFX / flash / overlays (ADR 0010).
// These are not user Assets. Local pack: system-media/<styleProfileId>/…
// Storage keys: system/style-profiles/<styleProfileId>/<id>
package systemmedia

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/doanbenson/momentos/services/render-service/internal/songpick"
)

// Item describes one packed system media file.
type Item struct {
	ID       string
	Filename string
	Kind     string // sfx | overlay
	MIME     string
}

const storageKeyPrefix = "system/style-profiles/" + songpick.StyleProfileID

var items = map[string]Item{
	"flash": {
		ID:       "flash",
		Filename: "flash.png",
		Kind:     "overlay",
		MIME:     "image/png",
	},
	"sfx-shutter": {
		ID:       "sfx-shutter",
		Filename: "sfx-shutter.wav",
		Kind:     "sfx",
		MIME:     "audio/wav",
	},
	"overlay-film-burn": {
		ID:       "overlay-film-burn",
		Filename: "overlay-film-burn.png",
		Kind:     "overlay",
		MIME:     "image/png",
	},
}

// Catalog returns a copy of known system media ids for this Style Profile.
func Catalog() map[string]Item {
	out := make(map[string]Item, len(items))
	for k, v := range items {
		out[k] = v
	}
	return out
}

// StorageKey is the object key convention (seed into storage like Edit Audio).
func StorageKey(mediaID string) string {
	return storageKeyPrefix + "/" + mediaID
}

// DefaultRoot prefers RENDER_SYSTEM_MEDIA_DIR, else SYSTEM_MEDIA_DIR, else "system-media".
func DefaultRoot() string {
	if v := os.Getenv("RENDER_SYSTEM_MEDIA_DIR"); v != "" {
		return v
	}
	if v := os.Getenv("SYSTEM_MEDIA_DIR"); v != "" {
		return v
	}
	return "system-media"
}

// Resolve returns the absolute path to a packed system media file under root.
func Resolve(root, mediaID string) (string, error) {
	item, ok := items[mediaID]
	if !ok {
		return "", fmt.Errorf("unknown system media id: %s", mediaID)
	}
	if root == "" {
		root = DefaultRoot()
	}
	path := filepath.Join(root, songpick.StyleProfileID, item.Filename)
	st, err := os.Stat(path)
	if err != nil || st.IsDir() || st.Size() == 0 {
		return "", fmt.Errorf("system media not found: %s (%s)", mediaID, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path, nil
	}
	return abs, nil
}

// ResolveAll resolves every catalog entry under root.
func ResolveAll(root string) (map[string]string, error) {
	out := make(map[string]string, len(items))
	for id := range items {
		p, err := Resolve(root, id)
		if err != nil {
			return nil, err
		}
		out[id] = p
	}
	return out, nil
}
