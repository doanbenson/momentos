import { useCallback, useEffect, useRef, useState } from "react";
import { getMoment } from "@/lib/moments";
import type { Moment, MomentStatus } from "@/lib/types";

const TERMINAL: MomentStatus[] = ["ready", "failed"];

type Options = {
  momentId: string | undefined;
  intervalMs?: number;
  enabled?: boolean;
};

export function useMomentPoll({
  momentId,
  intervalMs = 2000,
  enabled = true,
}: Options) {
  const [moment, setMoment] = useState<Moment | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [polling, setPolling] = useState(false);
  const alive = useRef(true);

  const refresh = useCallback(async () => {
    if (!momentId) return null;
    try {
      const next = await getMoment(momentId);
      if (!alive.current) return null;
      setMoment(next);
      setError(null);
      return next;
    } catch (e) {
      if (!alive.current) return null;
      setError(e instanceof Error ? e.message : String(e));
      return null;
    }
  }, [momentId]);

  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);

  useEffect(() => {
    if (!momentId || !enabled) return;

    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;

    async function tick() {
      setPolling(true);
      const next = await refresh();
      setPolling(false);
      if (cancelled || !next) {
        if (!cancelled) {
          timer = setTimeout(tick, intervalMs);
        }
        return;
      }
      if (TERMINAL.includes(next.status)) return;
      timer = setTimeout(tick, intervalMs);
    }

    void tick();

    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
    };
  }, [momentId, enabled, intervalMs, refresh]);

  return { moment, error, polling, refresh };
}

export function statusProgress(status: MomentStatus | undefined): number {
  switch (status) {
    case "intake":
      return 0.15;
    case "drafting":
      return 0.45;
    case "rendering":
      return 0.75;
    case "ready":
      return 1;
    case "failed":
      return 1;
    default:
      return 0.08;
  }
}

export function statusCopy(status: MomentStatus | undefined): {
  title: string;
  detail: string;
} {
  switch (status) {
    case "intake":
      return { title: "Queued", detail: "Preparing your Assets…" };
    case "drafting":
      return { title: "Drafting Cut", detail: "Aligning clips to the Beat Grid…" };
    case "rendering":
      return { title: "Rendering", detail: "Turning your Cut into a Moment…" };
    case "ready":
      return { title: "Ready", detail: "Your Moment is ready to refine." };
    case "failed":
      return {
        title: "Something failed",
        detail: "We couldn’t finish this Moment. Retry when ready.",
      };
    default:
      return { title: "Crafting your Moment", detail: "Starting…" };
  }
}
