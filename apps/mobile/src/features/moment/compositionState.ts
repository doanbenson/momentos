import type { MomentStatus } from "@/lib/types";

/** Non-terminal statuses while a Moment is still composing. */
export function isCompositionInProgress(
  status: MomentStatus | undefined,
): boolean {
  return (
    status === "intake" || status === "drafting" || status === "rendering"
  );
}

/** True when the Expo Router pathname is the Processing route. */
export function isOnProcessingRoute(pathname: string | null | undefined): boolean {
  if (!pathname) return false;
  return pathname.includes("processing");
}

/**
 * Auto-navigate to Editor only if the user is still on Processing when ready
 * (ADR 0018). Otherwise notify via toast — do not steal focus.
 */
export function shouldAutoNavigateToEditor(
  status: MomentStatus | undefined,
  pathname: string | null | undefined,
): boolean {
  return status === "ready" && isOnProcessingRoute(pathname);
}
