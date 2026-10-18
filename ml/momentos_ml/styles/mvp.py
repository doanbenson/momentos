"""Hard-coded MVP Style Profile + Vibe for Live A Life You Will Remember.

Song Pick / Edit Audio mirrors:
- apps/mobile/src/config/song-pick.ts (MVP_SONG_PICK)
- media-service config for Platform Song + Edit Audio object path
ml does not invent a second Song catalog — use momentos_ml.config.MVP_SONG_PICK_ID.

Beat-span / chorus-drop / system-media ids: momentos_ml.styles.song_pick +
system_media (parity: packages/contracts/mvp-constants.json).
"""

MVP_STYLE_PROFILE_ID = "life-on-film-v1"
MVP_VIBE_ID = "live-a-life-remember-v0"

MVP_STYLE = {
    "id": MVP_STYLE_PROFILE_ID,
    "pacing": "documentary-pulse",
    "transitions": ["hard-cut", "fade-soft"],
}

MVP_VIBE = {
    "id": MVP_VIBE_ID,
    "modulates": ["pace", "hit_density", "silence_tolerance"],
}
