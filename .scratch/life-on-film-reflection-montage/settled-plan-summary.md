# Settled plan summary — Life On Film PRD delta

Status: **PLAN APPROVED** (human confirmed shared understanding 2026-10-17).
Ready for Phase 2 `/to-spec`. Do not re-grill A–E locks or Rounds 1–3 unless a conflict appears in spec.

## Intent

Delta on completed `life-on-film-reflection-montage` (A–E): multi-asset regression, Higgsfield-inspired motion/look fidelity, J-cut timing, Asset cap 20, non-blocking Film Grain UX — one combined shippable slice.

## Product locks

- Song Pick: **Live A Life You Will Remember** @ 126 BPM (“The Nights” = style reference only).
- Cap: **20 total `assetIds` including Reflection**; Reflection-only compose still allowed.
- Place-all-once: every Asset in Cut ≥ once; fail draft if impossible; no silent drops / no multi-cycle reuse.
- Cut fields: keep `role`; add `act`, `motionPreset`, `visualStyle`, `transitionIn` (enums in ADR 0015).
- Act 3 look (Super 8 gate ~980×1306 on 1080×1920 `#0A0A0A`, 18fps, Kodak + grain): **all `act=3` clips**.
- Timing constants: `j_cut_pre_roll_sec=4.0` @ 18% vol; `drop_offset_after_speech_sec=0.25`.
- Banner: Moment `reflectionQuestion` on **render job**.
- Flash: 3-frame; markers + `transitionIn: white_flash`; coalesce duplicates.
- Mobile: non-blocking poller + pill + toast; Film Grain only; light Editor badge/preserve-fields.
- Acceptance: 1+8 in mp4; graph_test 1+6; soft ~90s for 1+15; 10m encode timeout.

## Implement order

1. **Contracts** — OpenAPI Cut fields, `maxItems: 20`, `mvp-constants` timing, codegen mirrors.
2. **Parallel** — ml ∥ render ∥ mobile.

## Doc index

| Doc | Path |
|---|---|
| Parent spec (A–E) | `.scratch/life-on-film-reflection-montage/spec.md` |
| Facts | `prd-delta-facts.md` |
| Settled table | `settled-decisions.md` |
| This summary | `settled-plan-summary.md` |
| Glossary | `GLOSSARY.md` (Act, Motion Preset, Visual Style) |
| ADRs | `docs/adr/0014`–`0021` |
