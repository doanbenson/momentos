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
    // musicId: "", // TODO
  },
} as const;
