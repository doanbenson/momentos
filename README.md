# Momentos

Hackathon MVP: turn a week’s photos, videos, and voice into a Life On Film–inspired **Moment**, then edit and share (Instagram Audio Match).

## Monorepo

| Path | Stack |
|---|---|
| `apps/mobile` | Expo React Native |
| `services/*` | Go microservices |
| `ml/` | Python Cut drafting |
| `packages/contracts` | OpenAPI |
| `claude/` + `*/claude/` | Agent context |

## Agent onboarding

Read [`CLAUDE.md`](./CLAUDE.md) → [`claude/context.md`](./claude/context.md) → [`GLOSSARY.md`](./GLOSSARY.md).

## Quick start

```bash
cp .env.example .env
pnpm install
# Go services (separate terminals): see scripts/dev.sh
# ML: cd ml && uv sync && uv run python -m momentos_ml.worker
pnpm mobile
```

## MVP Song Pick

Hard-coded: **Live A Life You Will Remember** (`apps/mobile/src/config/song-pick.ts`).
