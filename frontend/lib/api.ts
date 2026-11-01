export class ApiError extends Error {
  status: number;
  body: unknown;
  constructor(message: string, status: number, body?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

export type ApiConfig = { apiUrl: string; token: string };

export function trimSlash(u: string): string {
  return u.replace(/\/+$/, "");
}

/** Pick the API base URL: explicit config wins, else same host on port 8000. */
export function resolveApiUrl(configured: string | undefined, loc?: { protocol: string; hostname: string }): string {
  if (configured && configured.trim()) return trimSlash(configured.trim());
  if (loc) return `${loc.protocol}//${loc.hostname}:8000`;
  return "http://localhost:8000";
}

type Opts = { method?: string; token?: string; signal?: AbortSignal; okStatuses?: number[] };

/**
 * Fetch JSON and check the HTTP status. Non-2xx responses throw ApiError with the
 * backend's `{"error": "..."}` message, unless the status is listed in okStatuses
 * (e.g. 422 for consensus under quorum), in which case the body is returned.
 */
export async function apiFetch<T>(base: string, path: string, opts: Opts = {}): Promise<{ status: number; data: T }> {
  const headers: Record<string, string> = {};
  if (opts.token) headers.Authorization = `Bearer ${opts.token}`;
  let res: Response;
  try {
    res = await fetch(`${trimSlash(base)}${path}`, { method: opts.method ?? "GET", headers, signal: opts.signal });
  } catch (e) {
    if (e instanceof DOMException && e.name === "AbortError") throw e;
    throw new ApiError("Backend injoignable", 0);
  }
  let body: unknown = null;
  const text = await res.text();
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = null;
    }
  }
  if (res.ok || (opts.okStatuses ?? []).includes(res.status)) {
    return { status: res.status, data: body as T };
  }
  const msg = (body as { error?: string } | null)?.error ?? `Erreur HTTP ${res.status}`;
  throw new ApiError(msg, res.status, body);
}

/** Keeps only the latest request: starting a new one aborts the previous. */
export function latestOnly() {
  let ctl: AbortController | null = null;
  return {
    next(): AbortSignal {
      ctl?.abort();
      ctl = new AbortController();
      return ctl.signal;
    },
  };
}

/** Wraps an async fn so overlapping calls share the in-flight one (no double submit). */
export function singleFlight<A extends unknown[], R>(fn: (...a: A) => Promise<R>) {
  let p: Promise<R> | null = null;
  return (...a: A): Promise<R> => {
    if (!p) p = fn(...a).finally(() => (p = null));
    return p;
  };
}
