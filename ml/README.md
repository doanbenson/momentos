# momentos-ml

Python worker that **drafts and revises Cuts**. Does not run ffmpeg.

Agents emit **beat indices**; `snap_to_grid` owns milliseconds (ADR 0002).

```bash
cd ml
uv sync --extra dev
uv run pytest
uv run python -m momentos_ml.worker
```

## Planner modes

| `CUT_PLANNER` | Behavior |
|---|---|
| `gemini` (default) | Single-shot Gemini structured JSON → placements |
| `stub` | Pack Assets on successive beat holds (CI / no API key) |

## One-shot without broker

```bash
set CUT_PLANNER=stub
set EDIT_AUDIO_PATH=path\to\edit-audio.m4a
set CUT_DRAFT_JOB_JSON={"jobType":"cut-draft","momentId":"...","mode":"draft","assets":[{"id":"...","type":"photo","order":0}]}
uv run python -m momentos_ml.worker
```

See `claude/context.md` for env vars and layout.
