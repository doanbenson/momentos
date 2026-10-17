# media-service — context

## Boundary

**Owns:** Asset metadata, upload URLs to Supabase Storage, Edit Audio reference metadata, Platform Song metadata for the MVP Song Pick.  
**Does not own:** Moment/Cut logic, ffmpeg, a Momentos music catalog (there isn’t one).

## Flow

```
API (internal/api)
  → business (internal/business)
  → database (internal/database) + Supabase Storage
```

| Layer | Path | Responsibility |
|---|---|---|
| API | `internal/api` | Upload session, Asset CRUD, Song Pick meta read |
| Business | `internal/business` | Validate type/size, mint storage paths |
| Database | `internal/database` | Asset rows; Song Pick / Platform Song meta |

## Jobs / events

None owned. Other services read Asset IDs.

## Dependencies

- Supabase Storage + Postgres
- Hard-coded MVP Song Pick metadata (title, platform music id placeholder)
