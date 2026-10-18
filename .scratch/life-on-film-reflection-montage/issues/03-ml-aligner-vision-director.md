# (C) ml — aligner → vision → director → snap/validate

Status: resolved

Blocked by: 01

Workstream: C (parallel after A)

Parent spec: `.scratch/life-on-film-reflection-montage/spec.md`

## Comments

- 2026-10-17: Claimed by Track C agent — implementing aligner → vision → director → snap/validate at primary CI seam (stub draft → validated Cut).
- 2026-10-17: Resolved — stub draft → validated Cut with roles/captions/markers/hero_outro; stages align→vision→director→snap→validate; Song Pick constants shared via `styles/song_pick.py` (Track E).

## Goal

Evolve ml Cut drafting for Life On Film reflection + montage: align Reflection, score montage candidates, plan roles/captions/markers, snap to Beat Grid, validate hero_outro.

## Requirements

- Stage order: faster-whisper aligner (default) + LLM keep-segments → vision scoring → director/planner → snap → validate
- Emit beat indices; snap owns ms; librosa Beat Grid; Song Pick / Style constants for legal spans
- Persist captions + markers on Cut (re-render must not re-run Whisper)
- Stub planner produces valid role/caption/marker/hero_outro Cuts for CI
- Style Profile `life-on-film-v1`; Vibe `live-a-life-remember-v0`
- No product ffmpeg in ml

## Tests (primary CI seam)

- Stub draft → validated Cut (roles, captions, markers, hero_outro, snapped ms)
- Extend snap / cut_schema prior art; caches OK

## Out of scope

- Render Act look; system media packing; mobile intake UI
