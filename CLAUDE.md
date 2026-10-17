# Momentos — agent entry

Read these before changing code. Prefer the smallest file that answers the question.

## Always start here

- Global product + MVP rules + repo map: `claude/context.md`
- Product language (canonical nouns): `GLOSSARY.md`
- Human architecture overview: `docs/architecture.md`

## When editing a surface

| Surface | Open |
|---|---|
| Mobile app | `apps/mobile/claude/context.md` then `apps/mobile/claude/domain.md` |
| Gateway | `services/gateway/claude/context.md` then `services/gateway/claude/domain.md` |
| Moments / Cuts | `services/moment-service/claude/context.md` then `services/moment-service/claude/domain.md` |
| Assets / audio meta | `services/media-service/claude/context.md` then `services/media-service/claude/domain.md` |
| Render / ffmpeg | `services/render-service/claude/context.md` then `services/render-service/claude/domain.md` |
| Cut drafting (Python) | `ml/claude/context.md` then `ml/claude/domain.md` |
| API shapes | `packages/contracts/openapi.yaml` |

## Rules of engagement

1. Use glossary nouns (Moment, Cut, Asset, Song Pick, …) in code and docs.
2. Agents draft Cuts; Go `render-service` turns Cut → Render. Do not put ffmpeg in `ml/`.
3. MVP Song Pick is hard-coded — see `apps/mobile/src/config/song-pick.ts` and service config mirrors.
4. Service layout is **API → business → database**. Keep those layers distinct.
