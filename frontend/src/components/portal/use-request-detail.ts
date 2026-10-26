"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { RequestDetail } from "@/lib/portal/types";

interface DetailState {
  detail: RequestDetail | null;
  loading: boolean;
  notFound: boolean;
  reload: () => void;
}

/** Loads one request with its attachments, history and the actions the current user may take. */
export function useRequestDetail(id: string): DetailState {
  const [detail, setDetail] = useState<RequestDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [version, setVersion] = useState(0);

  useEffect(() => {
    let active = true;
    api
      .get<RequestDetail>(`/api/v1/portal/requests/${id}`)
      .then((res) => {
        if (!active) return;
        setDetail(res.data ?? null);
        setNotFound(!res.data);
      })
      .catch(() => active && setNotFound(true))
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, [id, version]);

  const reload = useCallback(() => setVersion((v) => v + 1), []);
  return { detail, loading, notFound, reload };
}
