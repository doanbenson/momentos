# (Bδ) Mobile — Asset cap 20 + non-blocking composition UX

Status: ready-for-agent

Blocked by: 06

Workstream: Bδ (parallel after Aδ)

Parent spec: `.scratch/life-on-film-reflection-montage/spec.md`  
Delta spec: `.scratch/life-on-film-reflection-montage/spec-delta.md`

## Goal

Raise Week Intake to 20 Assets (including Reflection) and make Moment composition non-blocking: global poller, CompositionStatus pill, ready toast, auto-nav only if still on Processing. Film Grain visual system only. Primary demo surface.

## Requirements

- Intake cap constant = 20 total Assets including Reflection; lockstep with OpenAPI / moment-service (ADR 0016)
- Reflection-only compose still allowed; exactly one Reflection still required
- Non-blocking UX per ADR 0018: global Moment job poller survives leave/minimize; pill for in-progress; toast on ready; auto-navigate to Editor only if user remains on Processing when ready
- Visual system: Momentos Film Grain only (not Obsidian Sunset / alternate Stitch)
- Light Editor: Reflection / role badges; preserve `act` / motion/look fields on local reorder **only if cheap** (seam 4 DEFER otherwise)
- Editor ops unchanged: preview / reorder / replace / remove / re-render; no Motion Preset / LUT / caption-word / beat-grid pickers
- Hard-coded Song Pick MVP unchanged

## Tests

- Cap 20 covered by thin CreateMoment contract (seam 3) + shared constant; no mandatory new RN harness
- Non-blocking UX: **manual acceptance checklist** — no device-farm CI

## Out of scope

- Device-farm / browser E2E as merge gate; ml drafting; render graph; music catalog; vibe picker
