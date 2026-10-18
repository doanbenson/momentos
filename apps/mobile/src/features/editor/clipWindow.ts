import type { CutClip } from "@/lib/types";

/** Source window when set, otherwise the Cut timeline span. */
export function clipDurationMs(clip: CutClip): number | null {
  if (
    typeof clip.sourceInMs === "number" &&
    typeof clip.sourceOutMs === "number" &&
    clip.sourceOutMs > clip.sourceInMs
  ) {
    return clip.sourceOutMs - clip.sourceInMs;
  }
  if (
    typeof clip.endMs === "number" &&
    typeof clip.startMs === "number" &&
    clip.endMs > clip.startMs
  ) {
    return clip.endMs - clip.startMs;
  }
  return null;
}

export function formatDurationMs(ms: number): string {
  const safe = Math.max(0, Math.round(ms));
  const totalSeconds = Math.floor(safe / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes > 0) {
    return `${minutes}:${seconds.toString().padStart(2, "0")}`;
  }
  const tenths = Math.floor((safe % 1000) / 100);
  return `${seconds}.${tenths}s`;
}
