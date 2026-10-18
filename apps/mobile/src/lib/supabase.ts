import AsyncStorage from "@react-native-async-storage/async-storage";
import { createClient } from "@supabase/supabase-js";

const url = process.env.EXPO_PUBLIC_SUPABASE_URL ?? "";
const publishable = process.env.EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY ?? "";

export const supabaseConfigured = Boolean(url && publishable);

/** Project ref from EXPO_PUBLIC_SUPABASE_URL (e.g. abcd.supabase.co → abcd). */
export const supabaseProjectRef = (() => {
  try {
    if (!url) return "";
    const host = new URL(url).hostname;
    const m = host.match(/^([a-z0-9-]+)\.supabase\.co$/i);
    return m?.[1] ?? "";
  } catch {
    return "";
  }
})();

export const supabase = createClient(
  url || "https://placeholder.supabase.co",
  publishable || "placeholder",
  {
    auth: {
      storage: AsyncStorage,
      autoRefreshToken: true,
      persistSession: true,
      detectSessionInUrl: false,
    },
  },
);
