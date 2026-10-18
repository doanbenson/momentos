import {
  PropsWithChildren,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { usePathname, useRouter } from "expo-router";
import type { Moment } from "@/lib/types";
import {
  isOnProcessingRoute,
  shouldAutoNavigateToEditor,
} from "./compositionState";
import { useMomentPoll } from "./useMomentPoll";

type MomentJobsContextValue = {
  activeMomentId: string | null;
  moment: Moment | null;
  error: string | null;
  polling: boolean;
  /** Register a Moment so polling survives leave/minimize. */
  trackMoment: (momentId: string) => void;
  refresh: () => Promise<Moment | null>;
  /** Ready toast pending when composition finished off Processing. */
  readyToastVisible: boolean;
  dismissReadyToast: () => void;
  openReadyMoment: () => void;
  openProcessing: () => void;
};

const MomentJobsContext = createContext<MomentJobsContextValue | null>(null);

function MomentJobsEffects({
  activeMomentId,
  moment,
  onReadyOffRoute,
  onAutoNavigated,
}: {
  activeMomentId: string | null;
  moment: Moment | null;
  onReadyOffRoute: (momentId: string) => void;
  onAutoNavigated: (momentId: string) => void;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const handledReadyKey = useRef<string | null>(null);

  useEffect(() => {
    if (!activeMomentId || !moment) return;
    if (moment.status !== "ready") {
      handledReadyKey.current = null;
      return;
    }
    // Wait for Expo Router pathname so we do not toast while still on Processing.
    if (!pathname) return;

    const key = `${activeMomentId}:ready`;
    if (handledReadyKey.current === key) return;

    if (shouldAutoNavigateToEditor(moment.status, pathname)) {
      handledReadyKey.current = key;
      onAutoNavigated(activeMomentId);
      router.replace({
        pathname: "/(editor)",
        params: { momentId: activeMomentId },
      });
      return;
    }

    if (!isOnProcessingRoute(pathname)) {
      handledReadyKey.current = key;
      onReadyOffRoute(activeMomentId);
    }
  }, [
    activeMomentId,
    moment,
    moment?.status,
    pathname,
    router,
    onReadyOffRoute,
    onAutoNavigated,
  ]);

  return null;
}

export function MomentJobsProvider({ children }: PropsWithChildren) {
  const [activeMomentId, setActiveMomentId] = useState<string | null>(null);
  const [readyToastVisible, setReadyToastVisible] = useState(false);
  const router = useRouter();

  const { moment, error, polling, refresh } = useMomentPoll({
    momentId: activeMomentId ?? undefined,
    enabled: Boolean(activeMomentId),
  });

  const trackMoment = useCallback((momentId: string) => {
    setActiveMomentId(momentId);
    setReadyToastVisible(false);
  }, []);

  const dismissReadyToast = useCallback(() => {
    setReadyToastVisible(false);
  }, []);

  const onReadyOffRoute = useCallback((_momentId: string) => {
    setReadyToastVisible(true);
  }, []);

  const onAutoNavigated = useCallback((_momentId: string) => {
    setReadyToastVisible(false);
  }, []);

  const openReadyMoment = useCallback(() => {
    if (!activeMomentId) return;
    setReadyToastVisible(false);
    router.push({
      pathname: "/(editor)",
      params: { momentId: activeMomentId },
    });
  }, [activeMomentId, router]);

  const openProcessing = useCallback(() => {
    if (!activeMomentId) return;
    router.push({
      pathname: "/(processing)",
      params: { momentId: activeMomentId },
    });
  }, [activeMomentId, router]);

  const value = useMemo(
    () => ({
      activeMomentId,
      moment,
      error,
      polling,
      trackMoment,
      refresh,
      readyToastVisible,
      dismissReadyToast,
      openReadyMoment,
      openProcessing,
    }),
    [
      activeMomentId,
      moment,
      error,
      polling,
      trackMoment,
      refresh,
      readyToastVisible,
      dismissReadyToast,
      openReadyMoment,
      openProcessing,
    ],
  );

  return (
    <MomentJobsContext.Provider value={value}>
      <MomentJobsEffects
        activeMomentId={activeMomentId}
        moment={moment}
        onReadyOffRoute={onReadyOffRoute}
        onAutoNavigated={onAutoNavigated}
      />
      {children}
    </MomentJobsContext.Provider>
  );
}

export function useMomentJobs() {
  const ctx = useContext(MomentJobsContext);
  if (!ctx) {
    throw new Error("useMomentJobs must be used within MomentJobsProvider");
  }
  return ctx;
}
