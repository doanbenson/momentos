-- media-service: Asset metadata + Song Pick / Platform Song / Edit Audio seed

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$ BEGIN
  CREATE TYPE asset_type AS ENUM ('photo', 'video', 'voice');
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  CREATE TYPE asset_status AS ENUM ('pending', 'ready', 'failed');
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS assets (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       TEXT NOT NULL,
  type          asset_type NOT NULL,
  status        asset_status NOT NULL DEFAULT 'pending',
  mime_type     TEXT NOT NULL,
  byte_size     BIGINT NOT NULL CHECK (byte_size >= 0),
  storage_key   TEXT NOT NULL UNIQUE,
  thumbnail_key TEXT,
  duration_ms   INT,
  width         INT,
  height        INT,
  checksum      TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS assets_user_id_idx ON assets (user_id);
CREATE INDEX IF NOT EXISTS assets_user_created_idx ON assets (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS assets_status_idx ON assets (status);

CREATE TABLE IF NOT EXISTS song_picks (
  id                      TEXT PRIMARY KEY,
  title                   TEXT NOT NULL,
  edit_audio_storage_key  TEXT NOT NULL,
  edit_audio_mime         TEXT NOT NULL,
  platform_provider       TEXT NOT NULL DEFAULT 'instagram',
  platform_display_name   TEXT NOT NULL,
  platform_music_id       TEXT, -- nullable until Meta library id known
  created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO song_picks (
  id,
  title,
  edit_audio_storage_key,
  edit_audio_mime,
  platform_provider,
  platform_display_name,
  platform_music_id
) VALUES (
  'live-a-life-you-will-remember',
  'Live A Life You Will Remember',
  'system/song-picks/live-a-life-you-will-remember/edit-audio',
  'audio/mp4',
  'instagram',
  'Live A Life You Will Remember',
  NULL
) ON CONFLICT (id) DO NOTHING;
