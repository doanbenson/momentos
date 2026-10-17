package business

import "github.com/doanbenson/momentos/services/media-service/internal/database"

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
