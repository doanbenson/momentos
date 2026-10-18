# Life On Film — Reflection + Montage

Status: ready-for-agent

Feature slug: `life-on-film-reflection-montage`

Seams confirmed (human):
1. Primary CI — ml stub draft → validated Cut
2. Second CI — render-service ffmpeg plan/graph
3. Thin CreateMoment / Week Intake contract
4. Editor revise path — DEFER / light only
5. Rejected as primary: greenfield `src/*.py`, persisted `master_timeline`, live Gemini/Whisper in CI, Edit Audio in git, device-farm E2E

---

## Problem Statement

A user completing Week Intake wants a Life On Film–inspired Moment: one spoken Reflection as the Act 1 hook, a beat-synced montage of their week’s photos and videos, captions from what they said, Act 2 energy markers (chorus drop, flashes, SFX), and a cohesive Act 3 hero outro — then a Render they can preview in the Editor and share. Today the pipeline treats Assets uniformly, OpenAPI Cuts have no roles/captions/markers, Week Intake does not require a Reflection or question, and render-service does not execute Act-specific look or marker-driven overlays.

## Solution

Evolve the existing Cut draft → Render path (not a parallel timeline stack). Week Intake requires one Reflection Asset plus a fixed `reflectionQuestion` on the Moment. The ml worker aligns the Reflection (faster-whisper default), scores montage candidates (vision), drafts a Cut with clip roles, caption cues, and markers snapped to the Song Pick Beat Grid, and validates a mandatory `hero_outro` on a legal 4-beat span. Go render-service alone runs ffmpeg: Act 1 VO + captions, Act 2 marker execution via system media, Act 3 grade / Super 8 / 18fps posterize. Song Pick stays “Live A Life You Will Remember”; Edit Audio is not in git. Mobile is the primary demo surface; Editor ops stay preview / reorder / replace / remove / re-render.

## User Stories

1. As a user, I want Week Intake to ask me a Reflection question, so that my Moment opens with a meaningful spoken hook.
2. As a user, I want to record one Reflection (talking-head video preferred, voice allowed), so that Act 1 has a clear source Asset.
3. As a user, I want to add photo and video Assets from my week alongside that Reflection, so that the montage has material to cut.
4. As a user, I want intake to refuse compose without a Reflection, so that I don’t get a Moment missing Act 1.
5. As a user, I want my Moment to remember the Reflection question I answered, so that captions and framing stay tied to that prompt.
6. As a user, I want the hard-coded Song Pick (“Live A Life You Will Remember”) applied automatically, so that I don’t browse a catalog for the MVP.
7. As a user, I want processing to finish quickly enough for a live demo (under ~60s for one Reflection + ~15 clips), so that I can show the product end-to-end.
8. As a user, I want Act 1 to play my Reflection with timed captions of what I said, so that the hook is readable without sound.
9. As a user, I want montage clips cut on musical beats, so that the Moment feels intentionally paced.
10. As a user, I want chorus-drop sync within about ±1 frame, so that Act 2 hits feel professional.
11. As a user, I want flash / SFX accents at planned markers, so that Act 2 has Life On Film energy without me editing them.
12. As a user, I want a hero outro as the final beat-aligned span, so that the Moment lands with a clear Act 3 button.
13. As a user, I want Act 3 to look cohesive (warm grade, Super 8 border, 18fps posterize, no stretch), so that the Render meets the ship bar.
14. As a user, I want a ready Render in the mobile Editor, so that I can preview before sharing.
15. As a user, I want to reorder, replace, or remove Assets and re-render, so that I can fix obvious mistakes without a full timeline editor.
16. As a user, I want to share via the OS sheet plus Instagram Audio Match checklist, so that Platform Song can replace Edit Audio legally.
17. As an engineer, I want OpenAPI to define Moment `reflectionQuestion` and Cut roles/captions/markers, so that every surface shares one contract.
18. As an engineer, I want Cut to be the only persisted edit plan, so that we never maintain a second `master_timeline` source of truth.
19. As an engineer, I want ephemeral planner JSON only as a debug DTO, so that agents can inspect planning without shipping it on the API.
20. As an engineer, I want ml to emit beat indices and snap to ms once, so that beat math stays deterministic (ADR 0002).
21. As an engineer, I want Beat Grid from librosa plus Song Pick / Style Profile constants for legal spans, so that we don’t invent a fake clock.
22. As an engineer, I want faster-whisper as the default Reflection aligner in ml, so that caption cues persist on the Cut for re-render without re-running Whisper.
23. As an engineer, I want vision scoring in ml before the director, so that weak clips are deprioritized before placement.
24. As an engineer, I want Act 2 markers planned in ml and executed in render-service, so that SFX/overlays are deterministic from the Cut.
25. As an engineer, I want SFX and overlays as system media (not intake Assets), so that users don’t upload flash frames or whooshes.
26. As an engineer, I want ffmpeg only in render-service, so that ml never encodes product Renders.
27. As an engineer, I want Style Profile `life-on-film-v1` with Vibe `live-a-life-remember-v0` unchanged, so that grammar bumps without vibe churn.
28. As an engineer, I want `CUT_PLANNER=stub` and cached beats/features allowed in CI, so that tests stay fast and offline.
29. As an engineer, I want a thin CreateMoment contract test, so that Reflection + question requirements can’t regress silently.
30. As an engineer, I want stub-draft → Cut schema tests as the primary CI seam, so that roles/captions/markers/hero_outro stay valid.
31. As an engineer, I want render-service graph/plan tests as the second CI seam, so that Act 1–3 filter behavior stays asserted.
32. As an engineer, I want optional CLI as an eng harness only, so that mobile remains the primary demo path.
33. As a demo operator, I want one Reflection + montage Assets to produce a shareable Render without Editor caption/LUT pickers, so that the MVP stays narrow.
34. As a product owner, I want glossary nouns (Moment, Cut, Asset, Reflection, Song Pick, …) used in contracts and code, so that language stays consistent.
35. As a future contributor, I want workstreams A→ then B∥C∥D∥E, so that contracts land before parallel surface work.

## Implementation Decisions

### Architecture

- Evolve the existing pipeline: Modules 1–3 live in `ml/`; Module 4 is Go render-service ffmpeg only. No greenfield parallel `src/*.py` product stack.
- Cut is the sole persisted edit plan. Ephemeral planner JSON is allowed as a debug/planning DTO only; never expose or persist OpenAPI `master_timeline` (or equivalent) as source of truth.
- Cross-service flow remains: mobile Week Intake → gateway → media-service (Assets) → moment-service (Moment + cut-draft job) → ml drafts Cut → moment-service persists Cut + render job → render-service writes Render → mobile Editor.
- Service layout stays API → business → database; glossary nouns are authoritative.

### Workstream split (implement order)

- **(A) Contracts first** — OpenAPI + Moment/Cut schema (+ codegen consumers) before parallel work.
- **(B) Mobile** — Week Intake Reflection + question (parallel after A).
- **(C) ml** — aligner → vision → director → snap/validate (parallel after A).
- **(D) render-service** — Act 1 / Act 2 / Act 3 ffmpeg behavior (parallel after A).
- **(E) System media + Song Pick constants** — SFX/overlays + beat/span constants (parallel after A).
- Parallelization rule: after A lands, B ∥ C ∥ D ∥ E.

### OpenAPI / schema

- Moment gains required (for this slice) `reflectionQuestion` (string); CreateMomentRequest accepts/requires it.
- Cut clips gain `role`: `reflection` | `montage` | `hero_outro` (in addition to existing `assetId`, `order`, `startMs`, `endMs`).
- Cut gains `captions[]`: `{ text, startMs, endMs }`.
- Cut gains `markers[]`: `{ type, atMs, optional payload }` (types include at least J-cut, chorus drop, flash; exact enum set defined in OpenAPI when implemented).
- Every Cut for this slice must include a `hero_outro` clip on a legal 4-beat span; validation fails if missing.
- camelCase in contracts; mirror in ml Pydantic Cut validator and mobile types after codegen.
- EditorPatch ops unchanged: reorder / replace / remove / reDraft / reRender only.

### Week Intake / Moment

- Reflection is a glossary role over a voice or talking-head video Asset (not a separate store).
- Week Intake requires exactly one Reflection Asset among intake Assets, plus the Reflection question string on the Moment.
- Talking-head preferred; voice Asset allowed.
- Song Pick remains hard-coded MVP id `live-a-life-you-will-remember` (title “Live A Life You Will Remember”); optional `songPickId` on create stays for later swap UI.
- Mobile is primary demo; CLI may exist as optional eng harness only.

### Song Pick / Beat Grid / Style

- Edit Audio is not stored in git; resolved via signed URL / env path as today.
- Beat Grid from librosa; agents emit beat indices; snap owns ms (ADR 0002).
- Legal 1/2/4-beat spans and chorus-drop constants live as Song Pick / Style Profile constants (ADR 0007), not a fake clock.
- Style Profile id: `life-on-film-v1`. Vibe id unchanged: `live-a-life-remember-v0`.

### ml drafting pipeline

- Order of stages: aligner (faster-whisper default + LLM assist for keep-segments) → vision scoring → director/planner → snap → validate.
- WhisperX optional upgrade only if quality demands it.
- Caption cues and markers must land on the persisted Cut so re-render does not re-run Whisper.
- Vision scoring runs in ml before director placement.
- Act 2: ml plans markers; render executes them.
- Stub planner path must produce role/caption/marker/hero_outro Cuts for CI (`CUT_PLANNER=stub`, cached beats/features OK).
- ml must not run product ffmpeg / encode Renders.

### render-service

- Sole home for ffmpeg graph execution.
- Act 1: Reflection picture/VO path + burned or overlaid captions from Cut cues.
- Act 2: execute markers (chorus drop sync target ±1 frame; flash/SFX via system media).
- Act 3: Style Profile look — warm grade, Super 8 gate/border, 18fps posterize, no stretch; hero_outro clip required.
- Grade/Super 8/18fps are Style Profile behavior in render, not free-form per-clip authoring UI.
- System media for SFX/overlays is not modeled as user Assets.

### Editor

- Existing ops only: preview, reorder, replace, remove, regenerate/re-render.
- No caption-word editor, beat-grid editor, or LUT picker in this slice.
- Reflection swap only if easy (light); do not expand Editor surface area (seam 4 deferred).

### Demo / acceptance

- Hard targets: e2e under ~60s for one Reflection + ~15 clips; chorus-drop sync within ±1 frame.
- Ship bar includes Act 3 visual cohesion on the Render.
- Stubs/caches OK for CI/dev.

### Testing seams (confirmed)

1. **Primary CI:** ml stub draft → OpenAPI-valid Cut (roles, captions, markers, mandatory hero_outro, snapped ms). Prior art: stub planner + snap + cut_schema tests.
2. **Second CI:** render-service plan validation + filter graph asserts Act 1–3 / marker / caption behavior. Prior art: ffmpeg graph tests.
3. **Thin contract:** CreateMoment / Week Intake requires Reflection + `reflectionQuestion` (Go and/or OpenAPI-level; not full RN E2E required).
4. **Deferred:** deep Editor revise coverage; light preserve-order tests only if cheap.

## Testing Decisions

### What makes a good test

- Assert external behavior at the highest confirmed seams: Cut shape after draft; ffmpeg plan/graph outputs/structure from a Cut; CreateMoment rejection/acceptance rules.
- Do not assert internal planner prompt text, model weights, or incidental private helpers.
- Prefer fixtures + stubs over live network, Gemini, or Whisper in CI.
- Use glossary nouns in test names and fixtures.

### Modules under test

- **ml:** stub planner + draft pipeline, snap-to-grid, Cut schema validation (roles/captions/markers/hero_outro).
- **render-service:** plan validation and filter-complex construction for Act 1–3 and markers.
- **moment-service (thin):** CreateMoment / complete-cut-draft contract behavior for `reflectionQuestion` and Cut persistence round-trip as needed for seam 3.
- **mobile:** no mandatory new RN test harness for MVP; contract coverage may be OpenAPI/codegen parity.

### Prior art

- `ml/tests/test_stub_planner.py`, `test_snap.py`, cut schema validation, `test_revise.py` (light only).
- `services/render-service` ffmpeg `graph_test.go` (photo/video/VO graph + plan validate).
- media-service upload allowlist tests for voice as related Asset-type prior art.
- moment-service and mobile currently lack automated tests; seam 3 introduces the thinnest useful CreateMoment coverage.

### CI policy

- `CUT_PLANNER=stub` and cached Beat Grid / features allowed.
- No commercial Edit Audio binary required in git for tests (mock bytes / fixtures).
- Demo timing and ±1 frame sync are acceptance targets; full device-farm E2E is out of CI scope.

## Out of Scope

- Greenfield parallel Python/Remotion timeline stack or `master_timeline` as persisted API truth.
- Commercial Edit Audio committed to the repository.
- Momentos-owned music catalog browse; Instagram Graph publish API.
- Vibe classification UI; Style Profile picker UI; caption-word / beat-grid / LUT Editor controls.
- Expanding Editor beyond preview / reorder / replace / remove / re-render.
- WhisperX as default (optional upgrade only).
- Auth/storage platform changes (remain Supabase + gateway).
- Product ffmpeg inside `ml/`.
- Device-farm or browser E2E as a merge gate.
- Auto Song Pick recommendation / swap UI beyond the hard-coded MVP field.

## Further Notes

- ADRs grounding this slice: 0001 (SKIP LOCKED jobs), 0002 (beat indices), 0003 (evolve existing pipeline), 0004 (Song Pick / Edit Audio not in git), 0005 (Cut sole edit plan), 0006 (ffmpeg only in render-service), 0007 (Beat Grid + constants), 0008 (Week Intake requires Reflection), 0009 (roles/captions/markers), 0010 (system media for SFX), 0011 (demo acceptance), 0012 (OpenAPI fields), 0013 (Style Profile v1 + faster-whisper).
- Implementation issues (optional trackers): `.scratch/life-on-film-reflection-montage/issues/01`–`05` for workstreams A–E; all `ready-for-agent`. A blocks parallel start of B–E.
- Phase 3 agents should not re-grill product decisions already locked above; implement against this spec and ADRs.

## Comments

- 2026-10-17: Seams 1–3 YES, 4 DEFER, rejected primaries NO — human confirmed; spec published `ready-for-agent`.
