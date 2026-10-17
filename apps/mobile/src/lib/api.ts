import { supabase } from "./supabase";

const gateway = process.env.EXPO_PUBLIC_GATEWAY_URL ?? "http://localhost:8080";

/** Thin fetch wrapper — prefer generated @momentos/sdk-ts once codegen runs. */
export async function gatewayFetch(path: string, init?: RequestInit) {
  const {
    data: { session },
  } = await supabase.auth.getSession();

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(init?.headers as Record<string, string> | undefined),
  };

  if (session?.access_token) {
    headers.Authorization = `Bearer ${session.access_token}`;
  }

  return fetch(`${gateway}${path}`, {
    ...init,
    headers,
  });
}

export function gatewayBaseUrl() {
  return gateway;
}

export async function readError(res: Response): Promise<string> {
  try {
    const text = await res.text();
    return text || res.statusText;
  } catch {
    return res.statusText;
  }
}
