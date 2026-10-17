package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *Store) GetSongPick(ctx context.Context, id string) (*SongPick, error) {
	const q = `
		SELECT id, title, edit_audio_storage_key, edit_audio_mime,
			platform_provider, platform_display_name, platform_music_id,
			created_at, updated_at
		FROM song_picks WHERE id = $1`
	sp, err := scanSongPick(s.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return sp, err
}

func (s *Store) ListSongPicks(ctx context.Context) ([]SongPick, error) {
	const q = `
		SELECT id, title, edit_audio_storage_key, edit_audio_mime,
			platform_provider, platform_display_name, platform_music_id,
			created_at, updated_at
		FROM song_picks
		ORDER BY id`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SongPick
	for rows.Next() {
		sp, err := scanSongPick(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sp)
	}
	return out, rows.Err()
}

func scanSongPick(row scannable) (*SongPick, error) {
	var sp SongPick
	err := row.Scan(
		&sp.ID, &sp.Title, &sp.EditAudioStorageKey, &sp.EditAudioMime,
		&sp.PlatformProvider, &sp.PlatformDisplayName, &sp.PlatformMusicID,
		&sp.CreatedAt, &sp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &sp, nil
}
