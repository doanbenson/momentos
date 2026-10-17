const gateway = process.env.EXPO_PUBLIC_GATEWAY_URL ?? "http://localhost:8080";

/** Thin fetch wrapper — prefer generated @momentos/sdk-ts once codegen runs. */
export async function gatewayFetch(path: string, init?: RequestInit) {
  return fetch(`${gateway}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });
}
