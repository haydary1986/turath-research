"use client";

import { useEffect } from "react";
import { create } from "zustand";
import { ApiError, api } from "@/lib/api";
import { useAuth, type AuthUser } from "@/lib/auth";
import type { Responsibility } from "@/lib/portal/activity";
import {
  SESSION_REFRESH_INTERVAL_MS,
  refreshDue,
  shouldSignOut,
} from "@/lib/portal/session-refresh";

interface MyAccessState {
  /** When, and for which token, the data below was last loaded. */
  last: { token: string; at: number } | null;
  responsibilities: Responsibility[];
  refresh: (token: string, force?: boolean) => Promise<void>;
  reload: () => Promise<void>;
}

/**
 * The signed-in user's current profile and data-entry responsibilities. The backend
 * authorizes with the database's current role, so the stored user (localStorage,
 * from the login response) is re-read from /auth/me: a promoted or demoted user sees
 * the right menus without signing out, and a rejected token signs out cleanly.
 */
export const useMyAccess = create<MyAccessState>((set, get) => ({
  last: null,
  responsibilities: [],
  refresh: async (token, force = false) => {
    if (!refreshDue(get().last, token, Date.now(), force)) return;
    set({ last: { token, at: Date.now() } });
    const [me, mine] = await Promise.allSettled([
      api.get<AuthUser>("/api/v1/auth/me"),
      api.get<Responsibility[]>("/api/v1/me/responsibilities"),
    ]);
    // A sign-out or another sign-in while the requests ran makes their answers stale.
    if (useAuth.getState().token !== token) return;
    if (me.status === "rejected") {
      if (me.reason instanceof ApiError && shouldSignOut(me.reason.status)) {
        useAuth.getState().logout();
        set({ last: null, responsibilities: [] });
      }
      return;
    }
    if (me.value.data) {
      // setAuth stores a fresh object; the previous user object is never modified.
      useAuth.getState().setAuth(token, { ...me.value.data });
    }
    if (mine.status === "fulfilled") {
      set({ responsibilities: mine.value.data ?? [] });
    }
  },
  reload: async () => {
    const token = useAuth.getState().token;
    if (token) await get().refresh(token, true);
  },
}));

/**
 * Keeps the session fresh while the portal shell is mounted: at once, every few
 * minutes, and when the window regains focus (throttled). Effects run only on the
 * client after hydration, so server and client markup stay identical.
 */
export function useSessionRefresh(): void {
  const token = useAuth((s) => s.token);
  const refresh = useMyAccess((s) => s.refresh);
  useEffect(() => {
    if (!token) return;
    void refresh(token);
    const timer = window.setInterval(() => void refresh(token), SESSION_REFRESH_INTERVAL_MS);
    const onFocus = () => void refresh(token);
    window.addEventListener("focus", onFocus);
    return () => {
      window.clearInterval(timer);
      window.removeEventListener("focus", onFocus);
    };
  }, [token, refresh]);
}

/** The caller's responsibilities (loaded by the shell's session refresh, or here on first use). */
export function useMyResponsibilities(): Responsibility[] {
  const token = useAuth((s) => s.token);
  const refresh = useMyAccess((s) => s.refresh);
  const responsibilities = useMyAccess((s) => s.responsibilities);
  useEffect(() => {
    if (token) void refresh(token);
  }, [token, refresh]);
  return token ? responsibilities : [];
}
