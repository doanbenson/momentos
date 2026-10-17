# render-service — context

## Boundary

**Owns:** Consuming render jobs, running ffmpeg, writing Render artifacts to storage, job status.  
**Does not own:** Cut drafting, Asset upload UX, Song selection.

## Flow

```
API (internal/api) — status / health
  → business (internal/business) — job lifecycle
  → database (internal/database) — job rows
  → ffmpeg (internal/ffmpeg) — encode
```

| Layer | Path | Responsibility |
|---|---|---|
| API | `internal/api` | Health + render status |
| Business | `internal/business` | Load Cut, fetch Assets, call Runner, persist result |
| Database | `internal/database` | Job + output metadata |
| ffmpeg | `internal/ffmpeg` | Only place encoding is invoked |

## Jobs / events

| Job | Consumer |
|---|---|
| `render` | render-service worker |

## Dependencies

- Cut JSON from moment-service
- Asset / Edit Audio URLs from media-service / Supabase Storage
- ffmpeg available on PATH in runtime image
