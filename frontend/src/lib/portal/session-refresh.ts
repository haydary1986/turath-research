/** How often the stored user is re-read from the backend while the portal is open. */
export const SESSION_REFRESH_INTERVAL_MS = 5 * 60 * 1000;
/** Focus events closer together than this do not refetch. */
export const SESSION_REFRESH_MIN_GAP_MS = 60 * 1000;

/** Whether a refresh is due: never loaded, another sign-in, a forced reload, or the gap has passed. */
export function refreshDue(
  last: { token: string; at: number } | null,
  token: string,
  now: number,
  force: boolean
): boolean {
  if (!last || last.token !== token || force) return true;
  return now - last.at >= SESSION_REFRESH_MIN_GAP_MS;
}

/** A 401 means the token is no longer accepted (expired, or the account was deleted): sign out. */
export function shouldSignOut(status: number | undefined): boolean {
  return status === 401;
}
