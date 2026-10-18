import {
  PropsWithChildren,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import type { Session } from "@supabase/supabase-js";
import { supabase, supabaseConfigured, supabaseProjectRef } from "./supabase";

type AuthContextValue = {
  session: Session | null;
  loading: boolean;
  configured: boolean;
  /** Last anonymous sign-in failure (e.g. provider disabled in Dashboard). */
  error: string | null;
  signInAnonymously: () => Promise<void>;
  signOut: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

function formatAnonError(error: { message: string; code?: string }): string {
  if (error.code === "anonymous_provider_disabled") {
    const providersUrl = supabaseProjectRef
      ? `https://supabase.com/dashboard/project/${supabaseProjectRef}/auth/providers`
      : "https://supabase.com/dashboard/project/_/auth/providers";
    return (
      "Anonymous sign-ins are disabled for this Supabase project. " +
      `Enable Authentication → Providers → Anonymous (${providersUrl}). ` +
      "OAuth is out of scope for the MVP."
    );
  }
  return error.message;
}

export function AuthProvider({ children }: PropsWithChildren) {
  const [session, setSession] = useState<Session | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const signInAnonymously = useCallback(async () => {
    if (!supabaseConfigured) {
      throw new Error(
        "Supabase is not configured. Set EXPO_PUBLIC_SUPABASE_URL and EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY.",
      );
    }
    setError(null);
    const { error: signInError } = await supabase.auth.signInAnonymously();
    if (signInError) {
      const message = formatAnonError(signInError);
      setError(message);
      throw new Error(message);
    }
  }, []);

  useEffect(() => {
    let mounted = true;

    (async () => {
      const { data } = await supabase.auth.getSession();
      if (!mounted) return;

      if (data.session) {
        setSession(data.session);
        setLoading(false);
        return;
      }

      // MVP: anonymous only — no OAuth / email password.
      if (supabaseConfigured) {
        try {
          await signInAnonymously();
        } catch {
          /* error state set in signInAnonymously */
        }
      }

      if (mounted) setLoading(false);
    })();

    const { data: sub } = supabase.auth.onAuthStateChange((_event, next) => {
      setSession(next);
      setLoading(false);
    });

    return () => {
      mounted = false;
      sub.subscription.unsubscribe();
    };
  }, [signInAnonymously]);

  const signOut = useCallback(async () => {
    const { error: signOutError } = await supabase.auth.signOut();
    if (signOutError) throw signOutError;
  }, []);

  const value = useMemo(
    () => ({
      session,
      loading,
      configured: supabaseConfigured,
      error,
      signInAnonymously,
      signOut,
    }),
    [session, loading, error, signInAnonymously, signOut],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
