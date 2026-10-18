# (B) Mobile — Week Intake Reflection + question

Status: resolved

Blocked by: 01

Workstream: B (parallel after A)

Parent spec: `.scratch/life-on-film-reflection-montage/spec.md`

## Goal

Mobile Week Intake requires one Reflection Asset and captures/sends `reflectionQuestion` when creating a Moment. Primary demo surface.

## Requirements

- Prompt the fixed Reflection question; persist on Moment via CreateMoment
- Require exactly one Reflection (talking-head preferred; voice allowed) before compose
- Continue hard-coded Song Pick MVP
- Editor ops unchanged (preview / reorder / replace / remove / re-render); Reflection swap only if easy
- No caption-word, beat-grid, or LUT picker UI

## Tests

- Relies on thin CreateMoment contract (seam 3); no mandatory new RN E2E harness

## Out of scope

- CLI as primary UX; catalog browse; vibe picker
