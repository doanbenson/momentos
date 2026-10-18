# (Dδ) render-service — motion/look, Act3-all, J-cut, flash

Status: ready-for-agent

Blocked by: 06

Workstream: Dδ (parallel after Aδ)

Parent spec: `.scratch/life-on-film-reflection-montage/spec.md`  
Delta spec: `.scratch/life-on-film-reflection-montage/spec-delta.md`

## Goal

Execute Cut → Render with Motion Presets / Visual Styles from the Cut, Act 3 look on all `act=3` clips, J-cut mix and drop-offset timing, 3-frame flash coalesce, and Act 1 prompt banner from the render job. ffmpeg only here.

## Requirements

- Consume Cut `act` / `motionPreset` / `visualStyle` / `transitionIn` (ADR 0015); ml chooses, render executes
- Act 3 look on **all** `act=3` clips: Super 8 gate ~980×1306 on 1080×1920 `#0A0A0A`, 18fps posterize path, Kodak + grain (not hero_outro-only)
- J-cut mix: Edit Audio at 18% for `j_cut_pre_roll_sec` before Act 1 picture end; honor `drop_offset_after_speech_sec` (ADR 0019)
- Flash: 3-frame white flash + shutter SFX; markers and `transitionIn: white_flash`; coalesce duplicates at same timestamp (ADR 0021)
- Act 1: Reflection question banner from render job `reflectionQuestion`; captions from Cut cues; voice-only slate path for `clean_916` Reflection (ADR 0020)
- System media remains catalog (not user Assets); Edit Audio resolution unchanged
- Soft acceptance targets (1+8 mp4, ~90s / 1+15, 10m encode timeout) are demo bars — not required as CI encode gates

## Tests (second CI seam)

- Plan validation + filter graph asserts for motion presets, Act3-all look, J-cut mix, flash coalesce, banner
- New/extended fixture: 1 Reflection + 6 montage (`graph_test`)
- Timing / Song Pick constants parity with contracts

## Out of scope

- ml place-all-once policy; Week Intake UI; device-farm E2E; Editor feature expansion
