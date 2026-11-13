"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { journalCheckParams, type JournalQuery } from "@/lib/portal/journal-query";
import type { JournalCheck } from "@/lib/portal/types";

export { journalCheckParams, type JournalQuery };

interface CheckState {
  requestId: string;
  result: JournalCheck | null;
  failed: boolean;
}

/** Runs the automatic check for the query, debounced and ignoring stale answers. */
export function useJournalCheck(query: JournalQuery, debounceMs = 700) {
  const key = journalCheckParams(query);
  const [attempt, setAttempt] = useState(0);
  const [state, setState] = useState<CheckState>({ requestId: "", result: null, failed: false });
  // The attempt is part of the identity, so a retry of the same query runs again.
  const requestId = key ? `${attempt}:${key}` : "";

  useEffect(() => {
    if (!requestId) return;
    let active = true;
    const timer = setTimeout(() => {
      api
        .get<JournalCheck>(`/api/v1/screening/check?${key}`)
        .then((res) => {
          if (active) setState({ requestId, result: res.data ?? null, failed: !res.data });
        })
        .catch(() => {
          if (active) setState({ requestId, result: null, failed: true });
        });
    }, debounceMs);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [key, requestId, debounceMs]);

  const retry = useCallback(() => setAttempt((current) => current + 1), []);
  const settled = state.requestId === requestId;
  return {
    result: key && settled ? state.result : null,
    failed: Boolean(key) && settled && state.failed,
    loading: Boolean(key) && !settled,
    retry,
  };
}
