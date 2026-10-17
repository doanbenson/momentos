# Momentos — global context

Hackathon MVP monorepo: Expo mobile + Go microservices + Python Cut drafting.

## Product (one paragraph)

Users complete **Week Intake** (up to 10 Assets: photo / video / voice), Momentos drafts a **Cut** in a Life On Film–inspired **Style Profile** modulated by a fixed **Vibe**, beat-synced to the hard-coded **Song Pick** (“Live A Life You Will Remember”), **Renders** it, then opens the **Editor** (preview, reorder, replace, remove, re-render). Users **save** in-app or **share** via OS sheet plus Instagram **Audio Match** checklist (Edit Audio for editing; Platform Song at Reel upload).

## MVP locks

- Song Pick: one hard-coded track; `song_pick` field exists for later auto-fill + swap UI
- Style Profile + Vibe: fixed pair; no vibe-classification UI
- Editor: preview / reorder / replace / remove / regenerate only
- Auth + DB + Storage: Supabase; domain writes go through **gateway**
- No Momentos music catalog; no Instagram Graph publish API

## Repo map

```
apps/mobile          Expo RN client
services/gateway     JWT edge + BFF
services/moment-service   Moment, Cut, jobs
services/media-service    Asset + Song Pick metadata
services/render-service   Cut → Render (ffmpeg)
ml/                  Python worker: draft/revise Cuts
packages/contracts   OpenAPI source of truth
```

## Cross-service flow

```
Week Intake (mobile)
  → gateway → media-service (Assets)
  → gateway → moment-service (create Moment, enqueue cut-draft)
  → ml worker (draft Cut)
  → moment-service (persist Cut)
  → render-service (Render)
  → Editor (mobile) → optional re-draft / re-render
  → save (Supabase via services) / share (OS + Audio Match checklist)
```

## Tooling

- **pnpm** + **Turborepo** (JS)
- **go.work** (Go services)
- **uv** (`ml/`)
- Codegen: `scripts/codegen.sh` from `packages/contracts`

## Language

Canonical nouns live only in root `GLOSSARY.md`. Service `claude/domain.md` files own which terms that service may write; they must not contradict the root glossary.
