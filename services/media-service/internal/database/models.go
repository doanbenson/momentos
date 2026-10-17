package database

import (
	"time"

	"github.com/google/uuid"
)

type AssetType string

const (
	AssetTypePhoto AssetType = "photo"
	AssetTypeVideo AssetType = "video"
	AssetTypeVoice AssetType = "voice"
)

type AssetStatus string

const (
	AssetStatusPending AssetStatus = "pending"
	AssetStatusReady   AssetStatus = "ready"
	AssetStatusFailed  AssetStatus = "failed"
)

type Asset struct {
	ID           uuid.UUID
	UserID       string
	Type         AssetType
	Status       AssetStatus
	MimeType     string
	ByteSize     int64
	StorageKey   string
	ThumbnailKey *string
	DurationMs   *int
	Width        *int
	Height       *int
	Checksum     *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type SongPick struct {
	ID                   string
	Title                string
	EditAudioStorageKey  string
	EditAudioMime        string
	PlatformProvider     string
	PlatformDisplayName  string
	PlatformMusicID      *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
