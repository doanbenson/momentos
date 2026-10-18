# render-service — context

## Boundary

**Owns:** Consuming render jobs, running ffmpeg, writing Render artifacts to storage, job status.  
**Does not own:** Cut drafting, Asset upload UX, Song selection.

## Flow

```
API (internal/api) — status / health
  → business (internal/business) — job lifecycle + in-process worker
  → database (internal/database) — job rows
  → ffmpeg (internal/ffmpeg) — encode (only place encoding is invoked)
```

| Layer | Path | Responsibility |
|---|---|---|
| Entrypoint | `cmd/render-service/main.go` | HTTP server + start in-process worker |
| API | `internal/api/mux.go` | `GET /healthz`, `GET /readyz`, job enqueue/status |
| Business | `internal/business/render.go` | Load Cut, fetch Assets, call Runner, persist result, retry ≤2 |
| Worker | `internal/business/worker.go` | In-process poll/claim (local Store; broker HTTP TODO) |
| Database | `internal/database/store.go` | Job + output metadata (in-memory MVP) |
| ffmpeg | `internal/ffmpeg/` | `Runner.Render` / `RenderFromCut`; one `filter_complex` graph |

## ffmpeg package map

| File | Role |
|---|---|
| `internal/ffmpeg/plan.go` | `Plan` / roles / act / motion / look / captions / markers; `life-on-film-v1` Validate |
| `internal/ffmpeg/graph.go` | Act 1 banner+captions+VO/slate; Act 2 J-cut mix + 3-frame flash coalesce; Act 3 look on all `act=3` |
| `internal/ffmpeg/runner.go` | Invokes local ffmpeg → H.264/AAC MP4; no placeholder on failure |

## Jobs / events

| Job | Consumer |
|---|---|
| `render` | render-service worker (`internal/business/worker.go`) |

## Config

| Env | Default | Purpose |
|---|---|---|
| `RENDER_SERVICE_ADDR` | `:8083` | HTTP listen |
| `RENDER_FFMPEG_PATH` | `ffmpeg` | Binary on PATH or absolute |
| `RENDER_EDIT_AUDIO_PATH` | _(empty)_ | MVP local Edit Audio fallback |
| `RENDER_SYSTEM_MEDIA_DIR` | `system-media` | Root containing `life-on-film-v1/` SFX/overlays |
| `RENDER_WORKER_CONCURRENCY` | `1` | Cap 1–2 |

## Dependencies

- Cut JSON + Asset type/URL map on job payload (moment-service broker HTTP still TODO)
- Edit Audio via job `editAudioUrl` or `RENDER_EDIT_AUDIO_PATH`
- System media (flash/SFX/overlays) via `internal/systemmedia` + pack under `system-media/` (see `scripts/seed-system-media.md`)
- Song Pick span/chorus constants: `internal/songpick` (parity: `packages/contracts/mvp-constants.json`)
- ffmpeg available on PATH (or `RENDER_FFMPEG_PATH`); readiness via `GET /readyz`
