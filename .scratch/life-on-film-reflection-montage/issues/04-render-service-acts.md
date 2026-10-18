# (D) render-service — Act 1 / 2 / 3 ffmpeg

Status: done

Blocked by: 01

Workstream: D (parallel after A)

Parent spec: `.scratch/life-on-film-reflection-montage/spec.md`

## Goal

Execute Cut → Render with Act-specific behavior: captions/VO, marker-driven accents, Act 3 Style Profile look. ffmpeg only here.

## Requirements

- Act 1: Reflection path + caption cues from Cut
- Act 2: execute markers (chorus drop ±1 frame target; flash/SFX via system media refs from E)
- Act 3: warm grade, Super 8 border, 18fps posterize, no stretch; honor hero_outro
- Consume Cut roles/captions/markers; no Whisper in render
- Edit Audio resolution unchanged (URL / env path; not from git)

## Tests (second CI seam)

- Plan validation + filter graph assertions for Act 1–3 / markers / captions
- Prior art: existing ffmpeg graph tests

## Out of scope

- ml planning; Week Intake UI; Editor feature expansion
