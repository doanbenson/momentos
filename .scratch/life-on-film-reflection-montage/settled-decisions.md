# Settled decisions — Life On Film PRD delta

Status: **PLAN APPROVED** (human confirmed 2026-10-17). Frontier closed.
Parent: `spec.md` (A–E locks stand). Summary: `settled-plan-summary.md`. Facts: `prd-delta-facts.md`.

## Round 1

| ID | Decision |
|---|---|
| R1-Q1 | Delta on existing slice; do not re-grill A–E unless conflict. |
| R1-Q2 | Song Pick = Live A Life @ 126 BPM; “The Nights” style-only. → ADR 0014 |
| R1-Q3 | One combined slice: ML + render + non-blocking mobile. |
| R1-Q4 | Acceptance-driven multi-asset regression; no greenfield ingest rewrite. |
| R1-Q5 | Contract-first motion/look on Cut. → ADR 0015 |
| R1-cap | Asset cap 20. → ADR 0016 |

## Round 2

| ID | Decision |
|---|---|
| R2-Q1 | Cap 20 = total `assetIds` including Reflection. |
| R2-Q2 | Place-all-once; fail if impossible; no silent drops / multi-cycle. → ADR 0017 |
| R2-Q3 | Keep `role` + add `act` 1\|2\|3. |
| R2-Q4 | `motionPreset` required on picture clips; four values + stub defaults. |
| R2-Q5 | `visualStyle` / `transitionIn` enums; planner-authored. |
| R2-Q6 | Act 3 look on all `act=3` clips. |
| R2-Q7 | J-cut 4.0s @ 18%; drop offset 0.25s. → ADR 0019 |
| R2-Q8 | Non-blocking UX (provider, minimize, pill, toast; auto-nav only if on Processing). → ADR 0018 |
| R2-Q9 | ~~Momentos Film Grain Stitch only.~~ **OVERRIDDEN 2026-10-18:** Moments Video Editor / Obsidian Sunset. |
| R2-Q10 | Gates 1+8 mp4, graph_test 1+6; ~90s soft for 1+15; 10m encode timeout. |

## Round 3

| ID | Decision |
|---|---|
| R3-Q1 | act defaults: reflection→1, hero_outro→3, else montage→2; Gemini may promote trailing montage to act=3 coda. → ADR 0020 |
| R3-Q2 | Act 1 prompt banner from Moment `reflectionQuestion` on render job. → ADR 0020 |
| R3-Q3 | Markers + `transitionIn: white_flash`; coalesce; 3-frame flash. → ADR 0021 |
| R3-Q4 | Voice-only Reflection: `clean_916` slate + banner + captions + VO; no motionPreset. → ADR 0020 |
| R3-Q5 | Light Editor fixes in slice (badges + preserve fields on reorder); no new ops. |
| R3-Q6 | Contracts first, then ml ∥ render ∥ mobile. |
| R3-Q7 | Reflection-only compose remains allowed. |

## ADR note

Prior scratch referenced ADRs 0001–0013 without files on disk. Delta decisions on disk: **0014–0021**.
