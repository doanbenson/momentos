/** Moments Video Editor — Obsidian Sunset (Stitch project). */
export const colors = {
  background: "#0B0B0E",
  surface: "#16161D",
  surfaceAlt: "#22222D",
  surfaceContainer: "#1F1F22",
  onSurface: "#F8FAFC",
  onSurfaceMuted: "#94A3B8",
  /** Sunset coral — primary CTA / playhead energy */
  primary: "#FF5722",
  primarySoft: "#FFB5A0",
  onPrimary: "#FFFFFF",
  /** Dusk violet — AI / secondary creative */
  secondary: "#E040FB",
  secondarySoft: "#FAABFF",
  /** Golden amber — warnings / badges */
  tertiary: "#FFB300",
  outline: "rgba(255, 255, 255, 0.07)",
  outlineStrong: "rgba(255, 255, 255, 0.12)",
  error: "#FFB4AB",
  glowPrimary: "rgba(255, 87, 34, 0.45)",
  glowSecondary: "rgba(224, 64, 251, 0.35)",
} as const;

export const radius = {
  sm: 8,
  md: 16,
  /** Soft media tiles */
  lg: 24,
  /** ROUND_FULL — pills, primary buttons, chips */
  full: 9999,
} as const;

export const space = {
  xs: 4,
  sm: 8,
  md: 16,
  lg: 24,
  xl: 32,
} as const;
