# PRD delta — facts (pre-grill)

Status: facts only — decisions pending grill rounds.
Source PRD: Higgsfield-inspired Life on Film ML + multi-asset render + non-blocking stitch UI.
Compared to: this scratch package (`spec.md`, issues 01–05).

## Existing scratch status

| Workstream | Issue | Status (tracker) |
|---|---|---|
| A Contracts | `issues/01-…` | resolved |
| B Mobile intake | `issues/02-…` | resolved |
| C ml | `issues/03-…` | resolved |
| D render | `issues/04-…` | done |
| E system media + constants | `issues/05-…` | done |

System media on disk: `services/render-service/system-media/life-on-film-v1/` (`flash.png`, `overlay-film-burn.png`, `sfx-shutter.wav`).

ADRs 0001–0013 are referenced in docs/scratch but `docs/adr/` is empty on disk (debt).

## Song Pick / rhythm

Source (ml): [Explore ML pipeline](5a1cd248-36fe-4087-8e7b-6478dc55bb2c).

- Canonical MVP Song Pick everywhere: id `live-a-life-you-will-remember`, title **Live A Life You Will Remember**, `NOMINAL_BPM` / `nominalBpm: 126`, `LEGAL_SPAN_BEATS=(1,2,4)`, `HERO_OUTRO_SPAN_BEATS=4`, `CHORUS_DROP_BEAT_INDEX=96`, Style/Vibe `life-on-film-v1` / `live-a-life-remember-v0`.
- **Not found in repo:** `"The Nights"`, `beat_1_sec`, `j_cut_pre_roll`, `drop_offset_after_speech_sec`.
- PRD `beat_*_sec` ≈ 0.476 / 0.952 / 1.905 = pure 60/126 math over legal 1/2/4-beat spans (already implied; not named `_sec` constants).
- `j_cut` is a **marker type**; stub director places near reflection end — **no** 4.0s pre-roll constant or render mix. Chorus energy is beat index `96`, not 0.25s post-speech offset.

## Multi-asset ingest / render

Sources: [Explore ML pipeline](5a1cd248-36fe-4087-8e7b-6478dc55bb2c), [Explore render-service](cfc34378-54e3-4215-ae00-f37c8610a90d).

- ml ingest **DONE** multi-asset: `extract_features` / `draft_cut` use the full Asset list (no `assets[0]` slice in ml).
- **CONFLICT with “sequence ALL assets”:** director + legacy stub_planner **drop trailing** when Beat Grid room + hero reserve insufficient; Gemini prompt allows dropping weak assets. Fallback uses `features[0]` only when **zero** placements.
- Outside ml: moment-service `assets[0]` hit is reflection-type validation on a single-id fetch, not montage ingest.
- render-service **DONE** multi-clip: `buildPlan` downloads every Cut clip; `graph.go` `concat=n=N` (1 clip → copy). Photo+video+VO test asserts `concat=n=2`.
- OpenAPI Cut: `role` / `captions` / `markers` / required `hero_outro` **DONE**; `act` / `motionPreset` / `visualStyle` / `transitionIn` **MISSING**.
- Acceptance “1 reflection + 8 visuals all in mp4” remains the regression gate; **MISSING** `graph_test` fixture for 1 reflection + 6 montage.
- Scratch issue D marked **done** against a narrower ship bar — **conflicts** with richer PRD checklist below.
- Primary CI seam already covered: `test_stub_draft_pipeline.py`, `test_stub_planner.py`, `test_cut_schema_reflection.py`, `test_song_pick_constants.py`.

## Camera / chromatic (gaps vs PRD)

Sources: [Explore ML pipeline](5a1cd248-36fe-4087-8e7b-6478dc55bb2c), [Explore render-service](cfc34378-54e3-4215-ae00-f37c8610a90d).

- ml `styles/mvp.py`: only `transitions: hard-cut, fade-soft` — **no** named camera presets.
- Photos in render: one hard-coded Ken Burns `zoompan` (z→1.12); videos scale/pad only. **MISSING** `hook_slow_push`, `whip_punch_in`, `handheld_super8_drift`, `native_action_crop` and Plan/Clip motion fields.
- Acts implied by roles in director (not schema `act`); Gemini prompt has no explicit 3-act grammar field.
- Act 1: 1080×1920 + `drawtext` captions + Reflection VO adelay/duck — **MISSING** prompt banner.
- Act 2: flash + `sfx-shutter` — **CONFLICT** duration ~**1 frame** (`oneFrameMs`), not 3-frame; `j_cut` marker type validated but **no J-cut mix / pre-roll** in `graph.go`.
- Act 3 (`hero_outro` + `life-on-film-v1`): warm `eq`, `drawbox` t=28, `fps=18`→30 posterize, hero letterbox — **MISSING** 4:3 gate ~980×1306 on `#0A0A0A` + grain; canvas stays 1080×1920.
- System media on disk: `flash.png`, `sfx-shutter.wav`, `overlay-film-burn.png` (latter two tiny placeholder bytes for flash/overlay).

## Mobile UX (gaps vs PRD)

Source: [Explore mobile UX](44b65269-130b-4cdf-9b3e-3060bae01e16) explore pass.

- Processing is **intentionally blocking**: `(processing)/index.tsx` comments “Non-dismissible until ready/failed”; Stack `gestureEnabled: false` / `headerBackVisible: false`; no minimize / continue-browsing control.
- Polling is **route-local** via `useMomentPoll` on `ProcessingScreen` only (2s until ready/failed).
- On `ready` → `router.replace("/(editor)")` — no ready toast / Snackbar.
- `CompositionStatus` exists as a **full-page** step list on Processing only — not a global pill.
- No `MomentJobsProvider` (root wraps `AuthProvider` only).
- Week Intake: **DONE** — `AssetFilmStrip` shows all Assets + Reflection badge; compose requires exactly one Reflection (`MAX_INTAKE_ASSETS = 10`).
- Editor: **PARTIAL** — `CutClipStrip` shows all clips; role/Reflection badge not shown; local reorder can drop `role` from local state.
- Stitch: app tokens match Stitch project **Momentos / Momentos Film Grain** (`#12100E`, `#C4A574`, `ROUND_FOUR`). Separate Stitch project **Moments Video Editor / Obsidian Sunset** (coral `#ff5722`, `ROUND_FULL` pills) does **not** match — grill later if PRD “Stitch alignment” meant that project.

## Implication for planning

This PRD is largely a **delta** on a mostly-landed Reflection+montage slice: Song identity conflict (name only; BPM already 126), motion/look fidelity + Cut clip fields, J-cut pre-roll / drop-offset constants, **must-place-every-Asset** policy (not `assets[0]` rewrite), render look upgrades, and non-blocking composition UX — not a greenfield re-plan of Acts/roles/captions/markers.

All three explore passes (ml / render / mobile) are merged above.
Rounds 1–3 settled; plan approved — see `settled-decisions.md`, `settled-plan-summary.md`, ADRs 0014–0021.
