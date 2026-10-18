# momentos-ml

Python worker that **drafts and revises Cuts**. Does not run ffmpeg.

Pipeline: align (faster-whisper / stub) → vision score → director → snap → validate.
Agents emit **beat indices**; `snap_to_grid` owns milliseconds (ADR 0002).

```bash
cd ml
uv sync --extra dev
# optional: uv sync --extra align   # faster-whisper for live Reflection align
uv run pytest
uv run python -m momentos_ml.worker
```

## Planner modes

| `CUT_PLANNER` | Behavior |
|---|---|
| `gemini` (default) | Gemini placements + aligner captions/markers |
| `stub` | Full Life On Film stub director (roles/captions/markers/hero_outro; CI) |

## One-shot without broker

```bash
set CUT_PLANNER=stub
set EDIT_AUDIO_PATH=path\to\edit-audio.m4a
set CUT_DRAFT_JOB_JSON={"jobType":"cut-draft","momentId":"...","mode":"draft","assets":[{"id":"...","type":"photo","order":0}]}
uv run python -m momentos_ml.worker
```

See `claude/context.md` for env vars and layout.
