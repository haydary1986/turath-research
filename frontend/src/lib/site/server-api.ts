/**
 * Server-side reads of the backend's public endpoints, for server components and
 * metadata routes. Responses are kept in Next's data cache for `revalidate` seconds,
 * so pages stay fast and the backend sees few requests.
 */

const BACKEND_URL = (process.env.BACKEND_URL || "http://localhost:8080").replace(/\/+$/, "");

export interface PageMeta {
  total: number;
  page: number;
  limit: number;
}

export interface BackendResult<T> {
  data: T;
  meta: PageMeta | null;
}

interface Envelope<T> {
  success: boolean;
  data?: T;
  error?: string;
  meta?: PageMeta;
}

/** Cache lifetimes (seconds) of the public reads. */
export const REVALIDATE = {
  content: 60,
  aggregates: 600,
  lists: 3600,
  entry: 21600,
  /** Sitemaps are fetched rarely and must reflect the lists as they are now. */
  fresh: 0,
} as const;

/**
 * GETs a public backend path. Returns null for a 404 (the thing does not exist) and
 * throws for anything else that is not a successful envelope.
 */
export async function backendGet<T>(path: string, revalidate: number): Promise<BackendResult<T> | null> {
  const res = await fetch(`${BACKEND_URL}${path}`, {
    headers: { Accept: "application/json" },
    // 0 means "always fresh" (sitemaps); anything else is kept in the data cache.
    ...(revalidate === 0 ? { cache: "no-store" as const } : { next: { revalidate } }),
  });
  if (res.status === 404) return null;
  let json: Envelope<T> | null = null;
  try {
    json = (await res.json()) as Envelope<T>;
  } catch {
    json = null;
  }
  if (!res.ok || !json?.success || json.data === undefined) {
    throw new Error(`backend GET ${path} failed: HTTP ${res.status}${json?.error ? ` (${json.error})` : ""}`);
  }
  return { data: json.data, meta: json.meta ?? null };
}

/**
 * Like backendGet, but a failure is logged and reported as null so an optional
 * section can be left out instead of failing the whole page.
 */
export async function backendGetOptional<T>(path: string, revalidate: number): Promise<BackendResult<T> | null> {
  try {
    return await backendGet<T>(path, revalidate);
  } catch (error: unknown) {
    console.error(error instanceof Error ? error.message : error);
    return null;
  }
}
