# (E) System media + Song Pick constants

Status: done

Blocked by: 01

Workstream: E (parallel after A)

Parent spec: `.scratch/life-on-film-reflection-montage/spec.md`

## Goal

Provide system media for SFX/overlays (not intake Assets) and Song Pick / Style Profile beat-span constants used by ml and render-service.

## Requirements

- System media packing/resolution for flash, SFX, overlays (not user Assets)
- Song Pick remains `live-a-life-you-will-remember`; no commercial stem in git
- Constants for legal 1/2/4-beat spans and chorus-drop timing shared/mirrored across surfaces as needed
- Style Profile `life-on-film-v1` constants available to render look + ml planning

## Tests

- Constant parity / resolution hooks as needed by seams 1–2; no Edit Audio binary in repo

## Out of scope

- Platform Song Meta API; catalog browse; vision/aligner logic
