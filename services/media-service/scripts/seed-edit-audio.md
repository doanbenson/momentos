# Seed Edit Audio into Supabase Storage

Canonical object (migration seed row):

- Bucket: `momentos-media` (or `MEDIA_STORAGE_BUCKET`)
- Key: `system/song-picks/live-a-life-you-will-remember/edit-audio`
- MIME: `audio/mp4` (m4a container is fine)
- Source (local preview mirror): `apps/mobile/assets/audio/live-a-life-you-will-remember.m4a`

## Prerequisites

1. Create a **private** bucket `momentos-media` in Supabase Storage (no public policies).
2. Run media-service once with `MEDIA_AUTO_MIGRATE=true` so `song_picks` row exists.
3. Have `SUPABASE_URL` + `SUPABASE_SECRET_KEY` available.

## Upload (curl)

```bash
# from repo root — adjust path if your m4a lives elsewhere
FILE=apps/mobile/assets/audio/live-a-life-you-will-remember.m4a
BUCKET=momentos-media
KEY=system/song-picks/live-a-life-you-will-remember/edit-audio

curl -X POST \
  "${SUPABASE_URL}/storage/v1/object/${BUCKET}/${KEY}" \
  -H "Authorization: Bearer ${SUPABASE_SECRET_KEY}" \
  -H "apikey: ${SUPABASE_SECRET_KEY}" \
  -H "Content-Type: audio/mp4" \
  -H "x-upsert: true" \
  --data-binary @"${FILE}"
```

## Verify via media-service

```bash
curl -s "${MEDIA_SERVICE_URL}/v1/song-picks/live-a-life-you-will-remember?sign=true" \
  -H "Authorization: Bearer ${INTERNAL_SERVICE_TOKEN}" \
  -H "X-Momentos-Principal: seed-check"
```

Expect `editAudioStorageKey` set and a non-empty `editAudioUrl` once the object exists.

`platformSong.musicId` stays `null` until the Meta library id is known.
