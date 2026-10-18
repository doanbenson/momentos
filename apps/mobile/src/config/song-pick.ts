/** MVP hard-coded Song Pick. Switcher UI comes later; field stays on Moment. */
export const MVP_SONG_PICK = {
  id: "live-a-life-you-will-remember",
  title: "Live A Life You Will Remember",
  /** Place licensed Edit Audio at this path when available. */
  editAudioPath: "assets/audio/live-a-life-you-will-remember.m4a",
  /** Instagram / Meta music library identity — fill when known for Audio Match checklist. */
  platformSong: {
    provider: "instagram" as const,
    displayName: "Live A Life You Will Remember",
    /** TODO: Meta music library id for checklist deep-link accuracy */
    musicId: undefined as string | undefined,
  },
} as const;

/**
 * Beat / Style constants for this Song Pick live in packages/contracts/mvp-constants.json
 * (mirrored by ml + render-service). Mobile keeps identity only for intake.
 */
export const MVP_STYLE_PROFILE_ID = "life-on-film-v1" as const;
