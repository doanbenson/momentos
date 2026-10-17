# moment-service — context

## Boundary

**Owns:** Moment lifecycle, Cut persistence, Style Profile / Vibe / Song Pick refs on the Moment, enqueue cut-draft and render jobs.  
**Does not own:** Asset bytes, ffmpeg, vibe ML training.

## Flow

```
API (internal/api)
  → business (internal/business)
  → database (internal/database) + jobs (internal/jobs)
```

| Layer | Path | Responsibility |
|---|---|---|
| API | `internal/api` | HTTP handlers for Moment/Cut |
| Business | `internal/business` | Intake → Moment, apply Editor commands, invalidate/reflow Cut |
| Database | `internal/database` | Moments, Cuts in Postgres (Supabase) |
| Jobs | `internal/jobs` | Enqueue `cut-draft` → ml; `render` → render-service |

## Jobs / events

| Job | Producer | Consumer |
|---|---|---|
| `cut-draft` | moment-service | `ml/` worker |
| `render` | moment-service | render-service |

## Dependencies

- Supabase Postgres
- media-service (Asset IDs must exist)
- Postgres job queue (`jobs` table, SKIP LOCKED) — ADR 0001
- media-service (Asset ready check + sign-download for job payloads)
- MVP Song Pick constants from env / shared config
- `INTERNAL_SERVICE_TOKEN`, `DATABASE_URL`
