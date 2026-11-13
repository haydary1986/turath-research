"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";

interface PageMeta {
  total: number;
  page: number;
  limit: number;
}

interface Loaded<T> {
  key: string;
  data: T | null;
  meta: PageMeta | null;
  error: string | null;
}

export interface PublicResource<T> {
  data: T | null;
  meta: PageMeta | null;
  /** The API error code/message, or null. */
  error: string | null;
  loading: boolean;
  reload: () => void;
}

/** Fetches a public API path; refetches when the path changes or reload() is called. */
export function usePublicResource<T>(path: string): PublicResource<T> {
  const [version, setVersion] = useState(0);
  const key = `${path}#${version}`;
  const [loaded, setLoaded] = useState<Loaded<T> | null>(null);

  useEffect(() => {
    let active = true;
    api
      .get<T>(path)
      .then((res) => {
        if (active) setLoaded({ key, data: res.data ?? null, meta: res.meta ?? null, error: null });
      })
      .catch((err: unknown) => {
        if (active) setLoaded({ key, data: null, meta: null, error: err instanceof Error ? err.message : "error" });
      });
    return () => {
      active = false;
    };
  }, [path, key]);

  const reload = useCallback(() => setVersion((v) => v + 1), []);
  const current = loaded?.key === key ? loaded : null;
  return {
    data: current?.data ?? null,
    meta: current?.meta ?? null,
    error: current?.error ?? null,
    loading: current === null,
    reload,
  };
}
