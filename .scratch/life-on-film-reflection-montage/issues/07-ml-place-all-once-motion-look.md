# (Cδ) ml — place-all-once + Cut motion/look + timing markers

Status: ready-for-agent

Blocked by: 06

Workstream: Cδ (parallel after Aδ)

Parent spec: `.scratch/life-on-film-reflection-montage/spec.md`  
Delta spec: `.scratch/life-on-film-reflection-montage/spec-delta.md`

## Goal

Evolve ml Cut drafting so every intake Asset is placed at least once, clips carry Act / Motion Preset / Visual Style / transitionIn, and markers honor shared J-cut / drop-offset constants. Stub path remains the primary CI seam.

## Requirements

- Place-all-once: every Moment Asset in Cut ≥ once; shorten to minimum legal Beat Grid spans before fail; no silent drops; no multi-cycle reuse; clear cut-draft failure if impossible (ADR 0017)
- Author `act`, `motionPreset`, `visualStyle`, `transitionIn` on clips; stub defaults deterministic for CI (ADR 0015)
- Act defaults: reflection→1, hero_outro→3, else montage→2; Gemini may promote trailing montage to act=3 coda (ADR 0020)
- Voice-only Reflection: `clean_916` slate + captions + VO; no Motion Preset; no borrowed montage photo (ADR 0020)
- Plan markers using `j_cut_pre_roll_sec` / `drop_offset_after_speech_sec` (ADR 0019); emit beat indices; snap owns ms
- Flash / transitionIn planning compatible with ADR 0021 coalesce rules on render side
- Song Pick / Style Profile / Vibe ids unchanged; no product ffmpeg in ml
- `CUT_PLANNER=stub` + cached beats/features OK for CI

## Tests (primary CI seam)

- Stub draft → validated Cut: place-all-once, new clip fields, markers/timing, hero_outro still required
- Failure path when placement impossible
- Extend stub draft / planner / cut schema / song-pick constant prior art

## Out of scope

- Render Act look / J-cut audio mix execution; mobile UX; CreateMoment cap enforcement (moment-service)
