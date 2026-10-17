package business

import (
	"context"
	"errors"
	"fmt"

	"github.com/doanbenson/momentos/services/media-service/internal/database"
)

type SongPickView struct {
	SongPick     *database.SongPick
	EditAudioURL string
}

func (s *Service) GetSongPick(ctx context.Context, id string, signEditAudio bool) (*SongPickView, error) {
	sp, err := s.store.GetSongPick(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.viewSongPick(ctx, sp, signEditAudio)
}

func (s *Service) ListSongPicks(ctx context.Context, signEditAudio bool) ([]SongPickView, error) {
	list, err := s.store.ListSongPicks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SongPickView, 0, len(list))
	for i := range list {
		v, err := s.viewSongPick(ctx, &list[i], signEditAudio)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, nil
}

func (s *Service) viewSongPick(ctx context.Context, sp *database.SongPick, sign bool) (*SongPickView, error) {
	v := &SongPickView{SongPick: sp}
	if !sign {
		return v, nil
	}
	url, err := s.storage.SignDownload(ctx, sp.EditAudioStorageKey, DownloadURLTTL)
	if err != nil {
		// Seed row may exist before Storage object is uploaded — return meta without URL.
		return v, nil
	}
	v.EditAudioURL = url
	return v, nil
}

// MVPSongPickID mirrors apps/mobile/src/config/song-pick.ts.
const MVPSongPickID = "live-a-life-you-will-remember"

func EnsureMVPSongPickSeeded(ctx context.Context, store *database.Store) error {
	_, err := store.GetSongPick(ctx, MVPSongPickID)
	if err == nil {
		return nil
	}
	if errors.Is(err, database.ErrNotFound) {
		return fmt.Errorf("song pick seed missing — run migrations: %w", err)
	}
	return err
}
