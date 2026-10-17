# gateway — context

## Boundary

**Owns:** HTTP edge for the mobile app, Supabase JWT validation, BFF aggregation.  
**Does not own:** Moment, Cut, Asset, or Render persistence.

## Flow

```
API (internal/api)
  → business (internal/business BFF orchestration)
  → database (none — peer HTTP + Supabase Auth only)
```

| Layer | Path | Responsibility |
|---|---|---|
| API | `internal/api` | Routes, auth middleware, request/response DTOs |
| Business | `internal/business` | Compose calls to moment / media / render |
| Database | `internal/database` | Empty — no domain DB |

## Jobs / events

None. Gateway is synchronous edge only.

## Dependencies

- Supabase Auth (JWT)
- `MOMENT_SERVICE_URL`, `MEDIA_SERVICE_URL`, `RENDER_SERVICE_URL`
- Contracts: `packages/contracts/openapi.yaml` (mobile-facing subset)
