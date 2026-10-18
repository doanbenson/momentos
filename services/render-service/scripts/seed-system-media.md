# Seed system media (SFX / flash / overlays)

System media is **not** user Assets (ADR 0010). Edit Audio stays separate (ADR 0004) and is never committed as a commercial stem.

## Local pack (dev / CI)

Checked into the repo as licensed-free procedural placeholders:

| Id | Path under `services/render-service/` |
|---|---|
| `flash` | `system-media/life-on-film-v1/flash.png` |
| `sfx-shutter` | `system-media/life-on-film-v1/sfx-shutter.wav` |
| `overlay-film-burn` | `system-media/life-on-film-v1/overlay-film-burn.png` |

Env (render-service prefers first):

- `RENDER_SYSTEM_MEDIA_DIR` — absolute/relative root that contains `life-on-film-v1/`
- `SYSTEM_MEDIA_DIR` — shared alias (also read by ml)

Default when unset: `system-media` (cwd-relative; module root when running from `services/render-service`).

Go resolve: `internal/systemmedia.Resolve(root, id)`.
Python catalog/resolve: `momentos_ml.styles.system_media`.

## Storage keys (Supabase, optional seed)

Mirror Edit Audio seeding style:

- Bucket: `momentos-media` (or `MEDIA_STORAGE_BUCKET`)
- Prefix: `system/style-profiles/life-on-film-v1/`
- Keys: `…/flash`, `…/sfx-shutter`, `…/overlay-film-burn`

Canonical ids + filenames: `packages/contracts/mvp-constants.json`.

```bash
# from repo root
BUCKET=momentos-media
ROOT=services/render-service/system-media/life-on-film-v1
PREFIX=system/style-profiles/life-on-film-v1

curl -X POST "${SUPABASE_URL}/storage/v1/object/${BUCKET}/${PREFIX}/flash" \
  -H "Authorization: Bearer ${SUPABASE_SECRET_KEY}" \
  -H "apikey: ${SUPABASE_SECRET_KEY}" \
  -H "Content-Type: image/png" \
  -H "x-upsert: true" \
  --data-binary @"${ROOT}/flash.png"

# repeat for sfx-shutter (audio/wav) and overlay-film-burn (image/png)
```

## Path convention for Track D (render ffmpeg)

1. Marker payload may reference `systemMediaId` (`flash` | `sfx-shutter` | `overlay-film-burn`).
2. Resolve via `systemmedia.Resolve(DefaultRoot(), id)` before building the filter graph.
3. Do not treat these paths as Cut `assetId`s.

Song Pick constants (legal 1/2/4-beat spans, chorus-drop beat index) live in `internal/songpick` and `momentos_ml.styles.song_pick`, parity-tested against `packages/contracts/mvp-constants.json`.
