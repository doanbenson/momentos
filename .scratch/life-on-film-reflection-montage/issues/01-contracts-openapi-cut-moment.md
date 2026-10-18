# (A) Contracts — OpenAPI Moment/Cut schema

Status: resolved

Blocked by: —

Workstream: A (must land before B∥C∥D∥E)

Parent spec: `.scratch/life-on-film-reflection-montage/spec.md`

## Goal

Extend OpenAPI (and generated/consumed mirrors) so Moment and Cut carry Reflection + montage fields; Cut remains sole persisted edit plan.

## Requirements

- Moment / CreateMomentRequest: `reflectionQuestion`
- Cut `clips[].role`: `reflection` | `montage` | `hero_outro`
- Cut `captions[]`: `text`, `startMs`, `endMs`
- Cut `markers[]`: `type`, `atMs`, optional payload
- Validation rule: every Cut for this slice requires `hero_outro` on a legal 4-beat span
- No persisted `master_timeline`
- Codegen / Pydantic / mobile types updated as needed for consumers
- Style Profile / Song Pick constants may be referenced but system media binaries are workstream E

## Tests

- Contract/schema validation for new fields and hero_outro rule (feeds seam 1 + 3)

## Out of scope

- Full ml pipeline, render graph, mobile UI beyond type sync
