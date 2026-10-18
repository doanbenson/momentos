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
# Local Postgres (matches DATABASE_URL in .env.example)
docker compose up -d postgres
# Go services (separate terminals): see scripts/dev.sh
# ML: cd ml && uv sync && uv run python -m momentos_ml.worker
pnpm mobile
```

## Test on your phone (Expo Go)

Fresh restart: stops any prior phone-stack processes, starts Postgres + gateway / moment / media / render / ML worker, points the app at your LAN IP, then opens Expo.

Requires **Docker Desktop** running (for Postgres on `localhost:5432`). Postgres data is kept across restarts.

```powershell
# Same Wi-Fi as your phone
pnpm phone

# Or Expo tunnel (if LAN/firewall blocks the device)
pnpm phone:tunnel
```

Fill `apps/mobile/.env` with Supabase URL + publishable key. In the Supabase Dashboard enable **Authentication → Providers → Anonymous** (OAuth is out of scope). Scan the QR code in Expo Go.

Expo Go SDK must match `apps/mobile` (`expo` ~57). If the store build lags, install a matching build from [expo.dev/go](https://expo.dev/go).

Stitch MCP: copy `mcp.json.example` → `.cursor/mcp.json` and add your key (`.cursor/` is gitignored).

## MVP Song Pick

Hard-coded: **Live A Life You Will Remember** (`apps/mobile/src/config/song-pick.ts`).
