package api

import (
	"time"

	"github.com/doanbenson/momentos/services/media-service/internal/business"
	"github.com/doanbenson/momentos/services/media-service/internal/database"
)

type assetDTO struct {
	ID           string  `json:"id"`
	UserID       string  `json:"userId"`
	Type         string  `json:"type"`
	Status       string  `json:"status"`
	MimeType     string  `json:"mimeType"`
	ByteSize     int64   `json:"byteSize"`
	StorageKey   string  `json:"storageKey"`
	ThumbnailKey *string `json:"thumbnailKey,omitempty"`
	DurationMs   *int    `json:"durationMs,omitempty"`
	Width        *int    `json:"width,omitempty"`
	Height       *int    `json:"height,omitempty"`
	Checksum     *string `json:"checksum,omitempty"`
	DownloadURL  string  `json:"downloadUrl,omitempty"`
	ThumbnailURL string  `json:"thumbnailUrl,omitempty"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}

func toAssetDTO(v *business.AssetView) assetDTO {
	a := v.Asset
	return assetDTO{
		ID:           a.ID.String(),
		UserID:       a.UserID,
		Type:         string(a.Type),
		Status:       string(a.Status),
		MimeType:     a.MimeType,
		ByteSize:     a.ByteSize,
		StorageKey:   a.StorageKey,
		ThumbnailKey: a.ThumbnailKey,
		DurationMs:   a.DurationMs,
		Width:        a.Width,
		Height:       a.Height,
		Checksum:     a.Checksum,
		DownloadURL:  v.DownloadURL,
		ThumbnailURL: v.ThumbnailURL,
		CreatedAt:    a.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:    a.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toAssetDTOFromAsset(a *database.Asset) assetDTO {
	return toAssetDTO(&business.AssetView{Asset: a})
}

type platformSongDTO struct {
	Provider    string  `json:"provider"`
	DisplayName string  `json:"displayName"`
	MusicID     *string `json:"musicId"` // nullable until Meta library id known
}

type songPickDTO struct {
	ID                   string          `json:"id"`
	Title                string          `json:"title"`
	EditAudioStorageKey  string          `json:"editAudioStorageKey"`
	EditAudioMime        string          `json:"editAudioMime"`
	EditAudioURL         string          `json:"editAudioUrl,omitempty"`
	PlatformSong         platformSongDTO `json:"platformSong"`
	// OpenAPI alias
	PlatformSongID *string `json:"platformSongId,omitempty"`
}

func toSongPickDTO(v *business.SongPickView) songPickDTO {
	sp := v.SongPick
	return songPickDTO{
		ID:                  sp.ID,
		Title:               sp.Title,
		EditAudioStorageKey: sp.EditAudioStorageKey,
		EditAudioMime:       sp.EditAudioMime,
		EditAudioURL:        v.EditAudioURL,
		PlatformSong: platformSongDTO{
			Provider:    sp.PlatformProvider,
			DisplayName: sp.PlatformDisplayName,
			MusicID:     sp.PlatformMusicID,
		},
		PlatformSongID: sp.PlatformMusicID,
	}
}
