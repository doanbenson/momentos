package business

import (
	"strings"

	"github.com/doanbenson/momentos/services/media-service/internal/database"
)

const (
	MiB = 1024 * 1024

	MaxPhotoBytes = 12 * MiB
	MaxVideoBytes = 80 * MiB
	MaxVoiceBytes = 8 * MiB

	// Light per-user create rate limit (hackathon floor).
	CreateRateLimitWindow = 60 // seconds
	CreateRateLimitMax    = 30
)

var mimeAllowlist = map[database.AssetType]map[string]struct{}{
	database.AssetTypePhoto: {
		"image/jpeg": {},
		"image/png":  {},
		"image/webp": {},
		// iPhone library default; ffmpeg may need libheif on some hosts.
		"image/heic": {},
		"image/heif": {},
	},
	database.AssetTypeVideo: {
		"video/mp4":       {},
		"video/quicktime": {},
	},
	database.AssetTypeVoice: {
		"audio/mp4":  {},
		"audio/m4a":  {},
		"audio/aac":  {},
		"audio/mpeg": {},
	},
}

func maxBytesFor(t database.AssetType) int64 {
	switch t {
	case database.AssetTypePhoto:
		return MaxPhotoBytes
	case database.AssetTypeVideo:
		return MaxVideoBytes
	case database.AssetTypeVoice:
		return MaxVoiceBytes
	default:
		return 0
	}
}

func mimeAllowed(t database.AssetType, mime string) bool {
	m, ok := mimeAllowlist[t]
	if !ok {
		return false
	}
	_, ok = m[mime]
	return ok
}

func normalizeContentType(ct string) string {
	ct = strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
	return ct
}

// contentTypeMatches accepts Storage Content-Type when it equals the declared
// mime, is allowlisted for the Asset type, or is a generic binary placeholder
// (common when mobile clients fail to stamp Content-Type on signed PUTs).
func contentTypeMatches(t database.AssetType, declared, stored string) bool {
	stored = normalizeContentType(stored)
	declared = normalizeContentType(declared)
	if stored == "" || stored == "application/octet-stream" {
		return true
	}
	if stored == declared {
		return true
	}
	return mimeAllowed(t, stored)
}
