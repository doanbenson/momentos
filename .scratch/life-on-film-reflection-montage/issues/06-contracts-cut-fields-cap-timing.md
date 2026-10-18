# (Aδ) Contracts — Cut motion/look fields, cap 20, timing constants

Status: resolved

Blocked by: —

Workstream: Aδ (must land before Cδ ∥ Dδ ∥ Bδ)

Parent spec: `.scratch/life-on-film-reflection-montage/spec.md`  
Delta spec: `.scratch/life-on-film-reflection-montage/spec-delta.md`

## Goal

Extend OpenAPI and shared constants so Cut clips carry Act / Motion Preset / Visual Style / transitionIn, CreateMoment accepts up to 20 Assets, and J-cut / drop-offset timing constants are mirrored for ml and render. Cut remains sole persisted edit plan.

## Requirements

- Cut clips: keep `role`; add `act` (`1`|`2`|`3`), `motionPreset` (four picture values), `visualStyle` (`clean_916`|`super8_gate`), `transitionIn` (`none`|`hard_cut`|`white_flash`) per ADR 0015
- CreateMoment / Week Intake `assetIds` `maxItems: 20` (total including Reflection) per ADR 0016
- Shared mvp-constants: `j_cut_pre_roll_sec=4.0`, `drop_offset_after_speech_sec=0.25` per ADR 0019; Song Pick stays Live A Life @ 126 (ADR 0014)
- Render job continues to carry Moment `reflectionQuestion` for Act 1 banner (ADR 0020) — document/contract as needed; do not duplicate onto Cut clips
- Codegen / Pydantic / Go / mobile type mirrors updated for consumers
- No persisted `master_timeline`

## Tests

- Schema / contract coverage for new Cut fields and maxItems 20 (feeds seams 1–3)
- Constants parity for new timing keys with existing Song Pick parity pattern

## Out of scope

- ml place-all-once logic, render graph execution, mobile non-blocking UI

## Answer

Shipped Aδ contracts:

- OpenAPI 0.0.4: Cut clip `act` / `motionPreset` / `visualStyle` / `transitionIn`; CreateMoment `maxItems: 20`; `RenderJobPayload.reflectionQuestion`; no `master_timeline`
- `mvp-constants.json`: `jCutPreRollSec=4.0`, `dropOffsetAfterSpeechSec=0.25` with ml + render-service parity tests
- Mirrors: Pydantic `CutClip`, moment-service cap + `ClipDTO`, render wire types + broker Job field, mobile types + `MAX_INTAKE_ASSETS=20`
- Snap fills role→Act/Motion/Visual defaults so stub→Cut CI stays green until issue 07 authors intentional values
- Picture-clip `motionPreset` required rule deferred to draft-time validation in 07 (needs Asset type); schema keeps it optional for voice-only Reflection

## Comments

- 2026-10-18: Implemented + code-reviewed; Status → resolved.
