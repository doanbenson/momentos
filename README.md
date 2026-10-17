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

## Test on your phone (Expo Go)

Starts gateway / moment / media / render / ML worker, points the app at your LAN IP, then opens Expo:

```powershell
# Same Wi-Fi as your phone
pnpm phone

# Or Expo tunnel (if LAN/firewall blocks the device)
pnpm phone:tunnel
```

Fill `apps/mobile/.env` with Supabase URL + anon key (anonymous sign-in enabled). Scan the QR code in Expo Go.

Stitch MCP: copy `mcp.json.example` → `.cursor/mcp.json` and add your key (`.cursor/` is gitignored).

## MVP Song Pick

Hard-coded: **Live A Life You Will Remember** (`apps/mobile/src/config/song-pick.ts`).
