# Life On Film — PRD delta (motion, place-all-once, cap 20, non-blocking UX)

Status: ready-for-agent

Feature slug: `life-on-film-reflection-montage` (delta on parent A–E)

Parent spec: `.scratch/life-on-film-reflection-montage/spec.md` (A–E locks stand; do not re-grill)

Seams confirmed (human, 2026-10-18):
1. Primary CI — ml stub draft → validated Cut — **YES** (place-all-once + motion/look + timing planning)
2. Second CI — render-service ffmpeg plan/graph — **YES** (Act3-all, J-cut, flash, motion, 1+6 fixture)
3. Thin CreateMoment / Week Intake contract — **YES** (cap 20 + Reflection rules)
4. Editor revise preserve fields — **DEFER** (light only if cheap during implement)
5. Mobile non-blocking UX — **Manual acceptance / no device-farm CI**
6. Rejected as primary: greenfield rewrites, `master_timeline`, live Gemini/Whisper in CI, Edit Audio in git, device-farm E2E, Obsidian Sunset Stitch, separate constants-only seam (fold timing into seams 1–2 + existing Song Pick / mvp-constants parity)

---

## Problem Statement

A user who already has Reflection + montage Moments wants the Life On Film Render to feel like a Higgsfield-inspired film: every Week Intake Asset appears at least once, camera Motion Presets and Visual Styles are intentional on the Cut, Act 3 look hits every Act 3 clip (not only the hero outro), J-cut and post-speech drop timing match the Song Pick grammar, Week Intake allows up to 20 Assets, and composition does not trap them on a blocking Processing screen. Today the landed A–E slice still drops trailing Assets under Beat Grid pressure, lacks Cut motion/look fields, hard-codes a single Ken Burns path, flashes for ~1 frame without J-cut mix semantics, caps intake at 10, and keeps Processing non-dismissible.

## Solution

Ship one combined delta on the existing Cut draft → Render path: raise the Asset cap to 20, require place-all-once drafting (fail clearly if impossible), extend Cut clips with Act / Motion Preset / Visual Style / transitionIn, plan J-cut and drop-offset constants into markers that render-service executes, apply Super 8 / Kodak / 18fps look to all `act=3` clips, burn the Reflection question banner from the render job, and make mobile composition non-blocking (global poller, pill, toast; Film Grain only). Song Pick remains “Live A Life You Will Remember” at 126 BPM; “The Nights” is style reference only. Parent A–E Reflection / roles / captions / markers / hero_outro / ffmpeg-only-in-render locks stand.

## User Stories

1. As a user, I want to add up to 20 Assets including my Reflection, so that a fuller week still fits in one Moment.
2. As a user, I want Reflection-only compose to still work, so that I can ship a hook without montage material.
3. As a user, I want every photo and video I added to appear at least once in the Cut, so that intake never silently discards my week.
4. As a user, I want drafting to fail with a clear error if placement is impossible, so that I am not surprised by a partial Moment.
5. As a user, I want montage clips to use named camera Motion Presets, so that the film feels intentional rather than one generic zoom.
6. As a user, I want Act 3 clips to share a cohesive Super 8 / Kodak / 18fps look, so that the ending reads as one Act, not only the final hero frame.
7. As a user, I want Edit Audio to J-cut under my Reflection before picture ends, so that the music lead-in feels cinematic.
8. As a user, I want montage energy to land shortly after my speech ends, so that Act 2 hits feel synced to what I said.
9. As a user, I want white-flash accents that last long enough to read (3 frames), so that Act 2 energy is visible.
10. As a user, I want the Reflection question visible as an Act 1 prompt banner, so that the hook is framed by what I answered.
11. As a user, I want voice-only Reflection to still show a clean slate with banner, captions, and VO, so that I am not forced to film myself.
12. As a user, I want composition to continue in the background after I leave Processing, so that I can keep browsing while the Moment cooks.
13. As a user, I want a CompositionStatus pill while work is in progress, so that I know the Moment is still alive.
14. As a user, I want a toast when the Moment is ready, so that I am notified without being yanked mid-browse.
15. As a user, I want auto-navigation to the Editor only if I am still on Processing when ready, so that background completion does not steal focus.
16. As a user, I want the Film Grain visual system for this UX, so that the product stays on-brand.
17. As a user, I want light Editor badges for Reflection / roles (and preserved motion/look on reorder if cheap), so that I can see what the Cut already decided.
18. As a user, I want a shareable Render that includes 1 Reflection + ~8 visuals in the mp4 for demo acceptance, so that multi-asset fidelity is proven.
19. As an engineer, I want OpenAPI Cut clips to carry `act`, `motionPreset`, `visualStyle`, and `transitionIn`, so that ml and render share one contract.
20. As an engineer, I want CreateMoment `assetIds` `maxItems: 20` enforced in OpenAPI, moment-service, and mobile together, so that the cap cannot drift.
21. As an engineer, I want shared Song Pick / Style constants for `j_cut_pre_roll_sec` and `drop_offset_after_speech_sec`, so that planning and mix stay aligned.
22. As an engineer, I want stub draft → validated Cut as the primary CI seam for place-all-once and motion/look fields, so that regressions fail fast offline.
23. As an engineer, I want render plan/graph tests as the second CI seam for Act3-all, J-cut mix, flash coalesce, and a 1+6 fixture, so that ffmpeg behavior stays asserted.
24. As an engineer, I want thin CreateMoment contract tests for the cap of 20, so that Week Intake limits cannot regress silently.
25. As an engineer, I want flash markers and `transitionIn: white_flash` to coalesce at the same timestamp, so that accents never double-fire.
26. As an engineer, I want Gemini allowed to promote trailing montage clips to `act=3` as a coda, while stubs keep role→act defaults, so that Act 3 look can extend beyond hero_outro when quality allows.
27. As an engineer, I want Cut to remain the sole persisted edit plan, so that we never introduce `master_timeline` as API truth.
28. As an engineer, I want ffmpeg only in render-service, so that ml never encodes product Renders.
29. As an engineer, I want `CUT_PLANNER=stub` and cached beats/features in CI, so that tests stay fast without live models.
30. As an engineer, I want contracts to land before parallel ml / render / mobile work, so that surfaces do not diverge.
31. As a demo operator, I want soft ~90s for 1+15 and a 10m encode timeout as acceptance targets, so that live demos stay honest without becoming CI gates.
32. As a product owner, I want glossary nouns (Moment, Cut, Asset, Reflection, Song Pick, Act, Motion Preset, Visual Style) used everywhere, so that language stays consistent.
33. As a future contributor, I want ADRs 0014–0021 to remain the decision record for this delta, so that implementers do not re-grill settled rounds.
34. As a mobile engineer, I want no mandatory device-farm E2E for non-blocking UX in this slice, so that CI stays lean while manual acceptance covers the UX.
35. As an Editor user, I want existing ops only (preview / reorder / replace / remove / re-render), so that this delta does not expand into a timeline tool.

## Implementation Decisions

### Architecture

- Delta on the completed Reflection + montage slice; evolve existing modules — do not greenfield a parallel drafting or timeline stack.
- Cut remains the sole persisted edit plan; ephemeral planner JSON may stay a debug DTO only; no OpenAPI `master_timeline`.
- Cross-service flow unchanged: Week Intake → gateway → media-service → moment-service → ml drafts Cut → moment-service persists Cut + render job → render-service → Editor.
- ffmpeg only in render-service; ml plans, never encodes product Renders.
- Service layout stays API → business → database; glossary nouns are authoritative.
- Parent A–E locks stand (Reflection required, roles/captions/markers, hero_outro, Style Profile `life-on-film-v1`, Vibe `live-a-life-remember-v0`, Song Pick Live A Life, Edit Audio not in git).

### Workstream split (implement order)

- **(Aδ) Contracts first** — OpenAPI Cut fields, CreateMoment `maxItems: 20`, mvp-constants timing, codegen / consumer mirrors.
- **(Cδ) ml** — place-all-once director/stub, author Act / Motion Preset / Visual Style / transitionIn, plan markers from timing constants (parallel after Aδ).
- **(Dδ) render-service** — execute motion/look, Act3-all look, J-cut mix, flash coalesce, banner from render job (parallel after Aδ).
- **(Bδ) mobile** — cap 20, non-blocking composition UX (Film Grain), light Editor badges / preserve fields if cheap (parallel after Aδ).
- Parallelization rule: after Aδ lands, Cδ ∥ Dδ ∥ Bδ.

### OpenAPI / schema

- Keep clip `role`: `reflection` | `montage` | `hero_outro`.
- Add clip `act`: `1` | `2` | `3`.
- Add clip `motionPreset` required on picture clips: `hook_slow_push` | `whip_punch_in` | `handheld_super8_drift` | `native_action_crop`.
- Add clip `visualStyle`: `clean_916` | `super8_gate`.
- Add clip `transitionIn`: `none` | `hard_cut` | `white_flash`.
- CreateMoment / Week Intake `assetIds` `maxItems: 20` (total including Reflection).
- Shared constants gain `j_cut_pre_roll_sec=4.0` and `drop_offset_after_speech_sec=0.25` (plus existing Song Pick / Beat Grid constants); mirrors stay in lockstep across contracts, ml, and render.
- Act 1 prompt banner text is Moment `reflectionQuestion` on the **render job** payload — not duplicated onto Cut clips.
- EditorPatch ops unchanged: reorder / replace / remove / reDraft / reRender only.
- camelCase in contracts; consumer mirrors updated after codegen.

### Week Intake / Moment

- Cap 20 = total `assetIds` including the single Reflection; enforce in OpenAPI, moment-service, and mobile together.
- media-service keeps per-Asset byte/rate limits only; gateway does not re-validate count.
- Reflection-only compose remains allowed.
- Song Pick remains hard-coded `live-a-life-you-will-remember` / “Live A Life You Will Remember” at nominal 126 BPM; “The Nights” is style/BPM reference only (ADR 0014).

### ml drafting

- Place-all-once: every Moment Asset appears in the Cut ≥ once; shorten holds to minimum legal Beat Grid spans before giving up; no silent trailing/weak drops; no multi-cycle Asset reuse in MVP; if still impossible, fail the cut-draft job with a clear error (ADR 0017).
- Stub/director Act defaults: `act=1` iff `role=reflection`, `act=3` iff `role=hero_outro`, other montage `act=2`; Gemini may promote trailing montage to `act=3` coda (ADR 0020).
- Author Motion Preset / Visual Style / transitionIn on clips; stub supplies deterministic defaults suitable for CI.
- Voice-only Reflection: `clean_916` slate + banner + captions + VO; no Motion Preset; do not borrow a montage photo (ADR 0020).
- Plan `j_cut` / `chorus_drop` (and related) markers using shared timing constants; snap still owns ms from beat indices.
- Persist captions, markers, and new clip fields on the Cut so re-render does not re-run Whisper or re-invent look.
- Stub planner path must produce place-all-once + motion/look Cuts for CI; `CUT_PLANNER=stub` and cached beats/features OK.

### render-service

- Execute Motion Presets and Visual Styles from Cut fields (ml chooses; render executes — ADR 0015).
- Act 3 look (Super 8 gate ~980×1306 on 1080×1920 canvas `#0A0A0A`, 18fps posterize path, Kodak + grain) applies to **all** `act=3` clips, not only `hero_outro`.
- J-cut mix: Edit Audio at 18% volume starting `j_cut_pre_roll_sec` before Act 1 picture end; honor `drop_offset_after_speech_sec` for post-speech energy (ADR 0019).
- Flash: 3-frame white flash + shutter SFX from system media; support markers and `transitionIn: white_flash`; coalesce duplicates at the same timestamp (ADR 0021).
- Act 1: draw Reflection question banner from render job; burn/overlay captions from Cut cues.
- System media remains non-Asset catalog entries; Edit Audio resolution unchanged (signed URL / env path).

### Mobile

- Raise intake cap constant to 20 in lockstep with contracts.
- Non-blocking composition: global Moment job poller survives minimize/continue; CompositionStatus pill; ready toast; auto-navigate to Editor only if user is still on Processing when ready (ADR 0018).
- Visual system: Momentos Film Grain only (not Obsidian Sunset / alternate Stitch project).
- Light Editor: role / Reflection badges; preserve new Cut fields on local reorder if cheap (seam 4 deferred otherwise).
- No caption-word, beat-grid, LUT, or Motion Preset picker UI in this slice.

### Demo / acceptance

- Hard CI: stub Cut validation; render graph_test fixture **1 Reflection + 6 montage**; CreateMoment cap 20.
- Manual / demo: **1 Reflection + 8 visuals present in mp4**; soft ~90s for 1+15; 10m encode timeout; non-blocking UX checklist on device.
- Chorus-drop sync target ±1 frame remains an acceptance bar (parent).

### Testing seams (confirmed)

1. **Primary CI:** ml stub draft → OpenAPI-valid Cut with place-all-once, Act / Motion Preset / Visual Style / transitionIn, markers informed by timing constants, clear failure when placement impossible. Prior art: stub draft / planner / cut schema / song-pick constant tests.
2. **Second CI:** render-service plan validation + filter graph asserts motion look, Act3-all, J-cut mix, 3-frame flash coalesce, banner consumption, and 1+6 fixture. Prior art: existing ffmpeg graph / plan tests; Song Pick / mvp-constants parity tests absorb new timing keys.
3. **Thin contract:** CreateMoment accepts ≤20 `assetIds` (including Reflection), rejects >20; Reflection + `reflectionQuestion` rules unchanged. Prior art: CreateMoment contract tests.
4. **Deferred:** deep Editor revise; light preserve-fields only if cheap.
5. **Manual only:** non-blocking mobile UX; full 1+8 encode acceptance; device-farm E2E out of CI.

## Testing Decisions

### What makes a good test

- Assert external behavior at the confirmed seams: Cut shape after stub draft; ffmpeg plan/graph structure from a Cut; CreateMoment acceptance/rejection for cap and Reflection rules.
- Do not assert internal planner prompt text, model weights, private helpers, or incidental filter string formatting beyond the behaviors above.
- Prefer fixtures + stubs over live network, Gemini, or Whisper in CI.
- Use glossary nouns in test names and fixtures.

### Modules under test

- **ml:** stub planner + draft pipeline, place-all-once policy, Cut schema validation for new clip fields, Song Pick / timing constant parity.
- **render-service:** plan validation and filter-complex construction for motion/look, Act3-all, J-cut, flash coalesce, banner; 1+6 graph fixture; constants parity with contracts.
- **moment-service (thin):** CreateMoment contract for max 20 Assets + existing Reflection rules.
- **mobile:** no mandatory new RN / device-farm harness; cap covered by seam 3 + shared constant; non-blocking UX is manual acceptance.

### Prior art

- Parent primary CI: stub draft pipeline, stub planner, cut schema reflection, song-pick constants.
- Parent second CI: render ffmpeg graph / plan validate tests; system media resolve; songpick constants parity with contracts.
- Parent seam 3: CreateMoment contract tests.
- Light revise helpers exist; extend only if preserving new fields is cheap (seam 4 deferred).

### CI policy

- `CUT_PLANNER=stub` and cached Beat Grid / features allowed.
- No commercial Edit Audio binary required in git for tests.
- No live Gemini / Whisper as merge gate.
- Demo timing, 1+8 mp4 presence, and non-blocking UX are acceptance targets — not device-farm CI gates.

## Out of Scope

- Re-grilling or rewriting parent A–E Reflection / roles / captions / markers / hero_outro foundation except where this delta explicitly extends them.
- Greenfield parallel Python/Remotion stack or persisted `master_timeline`.
- Commercial Edit Audio committed to the repository; Song Pick rename to “The Nights”; music catalog browse.
- Multi-cycle Asset reuse to fill music; silent dropping of intake Assets.
- Expanding Editor beyond preview / reorder / replace / remove / re-render (no Motion Preset / LUT / caption-word / beat-grid pickers).
- Device-farm or browser E2E as a merge gate; mandatory new RN test harness for non-blocking UX.
- Obsidian Sunset / alternate Stitch visual system.
- Instagram Graph publish API; auth/storage platform changes.
- Product ffmpeg inside ml.
- WhisperX as default (optional upgrade only, unchanged from parent).

## Further Notes

- ADRs for this delta: 0014 (Song Pick identity vs The Nights reference), 0015 (Cut motion/look fields), 0016 (cap 20), 0017 (place-all-once), 0018 (non-blocking UX), 0019 (J-cut / drop timing constants), 0020 (Act defaults + prompt banner + voice Reflection slate), 0021 (flash markers + transitionIn coalesce). Parent ADRs 0001–0013 remain in force where not superseded.
- Settled plan inputs: `settled-plan-summary.md`, `settled-decisions.md`, `prd-delta-facts.md`.
- Implementation issues: `06` contracts → then `07` ml ∥ `08` render ∥ `09` mobile; all `ready-for-agent`. `06` blocks parallel start of 07–09.
- Phase 3 agents should not re-grill product decisions locked above; implement against this delta, the parent spec, and ADRs 0014–0021.

## Comments

- 2026-10-18: Seams 1–3 YES, 4 DEFER, 5 manual/no device-farm, rejected primaries NO — human confirmed; `spec-delta.md` published `ready-for-agent`.
