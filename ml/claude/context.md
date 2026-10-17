# ml — context

## Boundary

**Owns:** Consuming `cut-draft` jobs, drafting/revising Cut JSON using agents + fixed Style/Vibe/Song Pick.  
**Does not own:** ffmpeg, Asset storage, HTTP BFF.

## Flow

```
worker (momentos_ml/worker)
  → agents (momentos_ml/agents)
  → styles (momentos_ml/styles) + contracts types
  → POST Cut back to moment-service (or job completion payload)
```

| Layer | Path | Responsibility |
|---|---|---|
| Worker | `momentos_ml/worker` | Queue consume / ack |
| Agents | `momentos_ml/agents` | Draft / revise Cut |
| Styles | `momentos_ml/styles` | MVP fixed Style Profile + Vibe |
| Contracts | `momentos_ml/contracts` | Generated OpenAPI types |

## Jobs / events

| Job | Action |
|---|---|
| `cut-draft` | `agents.draft_cut` → persist via moment-service |

## Dependencies

- Job broker (`JOB_QUEUE_URL`)
- Asset metadata (from job payload or media-service)
- moment-service completion API
