import * as React from "react"

const MOBILE_BREAKPOINT = 768
const QUERY = `(max-width: ${MOBILE_BREAKPOINT - 1}px)`

function subscribe(onChange: () => void): () => void {
  const mql = window.matchMedia(QUERY)
  mql.addEventListener("change", onChange)
  return () => mql.removeEventListener("change", onChange)
}

const isMobileNow = (): boolean => window.matchMedia(QUERY).matches
// The server renders the desktop layout; the client corrects it on hydration.
const isMobileOnServer = (): boolean => false

/** Whether the viewport is narrower than the mobile breakpoint, kept in sync with resizes. */
export function useIsMobile(): boolean {
  return React.useSyncExternalStore(subscribe, isMobileNow, isMobileOnServer)
}
