# Architecture (MVP)

## Surfaces

| Surface | Role |
|---|---|
| `apps/mobile` | Week Intake → processing → Editor → save/share |
| `services/gateway` | Supabase JWT + BFF |
| `services/moment-service` | Moment, Cut, jobs |
| `services/media-service` | Asset + Song Pick metadata |
| `services/render-service` | Cut → Render (ffmpeg) |
| `ml/` | Cut drafting worker |

## Sequence

```
mobile Week Intake
  → gateway → media-service (Assets)
  → gateway → moment-service (Moment + cut-draft job)
  → ml drafts Cut
  → moment-service stores Cut + render job
  → render-service writes Render
  → mobile Editor / Share
```

## Agent docs

Start at root `CLAUDE.md` → `claude/context.md` → per-surface `claude/context.md` + `domain.md`.
