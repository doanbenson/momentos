# mobile — context

## Boundary

**Owns:** Week Intake UI, processing wait state, Editor, save/share + Audio Match checklist, Supabase Auth client.  
**Does not own:** Cut drafting, ffmpeg, domain persistence (talks to gateway).

## Flow

```
Screens (app/*)
  → features (src/features/*)
  → lib (src/lib api + supabase)
  → gateway HTTP
```

| Layer | Path | Responsibility |
|---|---|---|
| UI routes | `app/(onboarding\|processing\|editor\|share)` | Expo Router screens |
| Features | `src/features/*` | Intake, editor, moment, share logic |
| Lib | `src/lib` | Supabase client, gateway fetch |
| Config | `src/config/song-pick.ts` | Hard-coded MVP Song Pick |

## MVP screen flow

1. Onboarding — Week Intake (≤10 Assets)
2. Processing — cut-draft + render
3. Editor — preview / reorder / replace / remove / re-render
4. Share — save or OS share + Audio Match checklist

## Dependencies

- Expo Router, Supabase JS
- `EXPO_PUBLIC_GATEWAY_URL`, `EXPO_PUBLIC_SUPABASE_*`
- Later: `@momentos/sdk-ts` from codegen
