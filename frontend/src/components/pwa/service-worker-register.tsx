"use client";

import { useEffect } from "react";

/** Registers /sw.js in production builds only; development never runs a service worker. */
export function ServiceWorkerRegister() {
  useEffect(() => {
    if (process.env.NODE_ENV !== "production" || !("serviceWorker" in navigator)) return;
    navigator.serviceWorker
      .register("/sw.js", { scope: "/", updateViaCache: "none" })
      .catch((error: unknown) => {
        // Offline support is an enhancement: the site works without it.
        console.warn("Service worker registration failed", error);
      });
  }, []);
  return null;
}
