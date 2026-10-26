import { useSyncExternalStore } from "react";

const subscribe = () => () => undefined;

/** False during server rendering and the hydration pass, true afterwards. */
export function useHydrated(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => true,
    () => false
  );
}
