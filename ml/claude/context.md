# ml — context

## Boundary

**Owns:** Consuming `cut-draft` jobs, drafting/revising Cut JSON using agents + fixed Style/Vibe/Song Pick.  
**Does not own:** ffmpeg, Asset storage, HTTP BFF.

## Flow

```
worker (momentos_ml/worker)
  → features (metadata AssetFeature)
  → align (faster-whisper default; stub in CI)
  → vision (score montage candidates; stub in CI)
  → beat (BeatGrid via librosa)
  → director / agents (stub | gemini → beat indices + roles/markers)
  → snap (indices → ms; captions offset; markers → atMs)
  → validate (OpenAPI Cut)
  → POST complete/fail → moment-service
```

| Layer | Path | Responsibility |
|---|---|---|
| Worker | `momentos_ml/worker` | Claim / process / complete cut-draft |
| Features | `momentos_ml/features` | Metadata-only AssetFeature extraction |
| Align | `momentos_ml/align` | Reflection word cues + keep-segments |
| Vision | `momentos_ml/vision` | Montage candidate scores before director |
| Beat | `momentos_ml/beat` | Edit Audio → BeatGrid; placements → ms |
| Agents | `momentos_ml/agents` | Director + draft / revise Cut (stub or Gemini) |
| Validate | `momentos_ml/validate` | Pydantic Cut aligned with OpenAPI |
| Styles | `momentos_ml/styles` | MVP Style Profile + Vibe + Song Pick span constants + system media catalog |
| Config | `momentos_ml/config.py` | Env + MVP Song Pick id |

## Jobs / events

| Job | Action |
|---|---|
| `cut-draft` (`mode=draft`) | `agents.draft_cut` → complete |
| `cut-draft` (`mode=revise`) | `agents.revise_cut` (constrained) → complete |

## Env

| Var | Default | Role |
|---|---|---|
| `CUT_PLANNER` | `gemini` | `gemini` \| `stub` |
| `GEMINI_API_KEY` | — | Required for gemini |
| `GEMINI_MODEL` | `gemini-2.0-flash` | Model id |
| `MOMENT_SERVICE_URL` | — | Base for `/v1/internal/jobs/claim` + complete |
| `INTERNAL_SERVICE_TOKEN` | — | Bearer for internal job routes |
| `JOB_QUEUE_URL` | — | Unused when moment-service jobs are primary |
| `EDIT_AUDIO_PATH` | — | Local Edit Audio when job has no URL |
| `SYSTEM_MEDIA_DIR` | render-service pack | Optional override for system media root |
| `BEAT_CACHE_DIR` | `.beat_cache` | BeatGrid disk cache |
| `CUT_DRAFT_JOB_JSON` | — | One-shot job JSON (no broker) |
| `WORKER_POLL_INTERVAL_S` | `2.0` | Idle poll |

## Dependencies

- Job broker / moment-service completion API (routes TBD — hooks in `worker/jobs.py`)
- Asset metadata on job payload (duration/dims; OpenAPI AssetRef is id+type only)
- Edit Audio bytes (job `editAudioUrl` or `EDIT_AUDIO_PATH`)
