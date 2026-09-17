import { useCallback, useSyncExternalStore } from "react";

const noopSubscribe = () => () => {};

const serverFalse = () => false;

/**
 * Subscribes to a CSS media query. Returns `false` during SSR and on the
 * hydrating render, then the real value once mounted on the client.
 */
export function useMediaQuery(query: string) {
  const subscribe = useCallback(
    (onChange: () => void) => {
      if (typeof window === "undefined" || !window.matchMedia) {
        return () => {};
      }
      const mql = window.matchMedia(query);
      mql.addEventListener("change", onChange);
      return () => mql.removeEventListener("change", onChange);
    },
    [query],
  );

  const getSnapshot = useCallback(() => {
    if (typeof window === "undefined" || !window.matchMedia) return false;
    return window.matchMedia(query).matches;
  }, [query]);

  return useSyncExternalStore(subscribe, getSnapshot, serverFalse);
}

/** True when the app is running as an installed PWA. */
export function useIsStandalone() {
  return useMediaQuery("(display-mode: standalone)");
}

interface WindowWithMSStream extends Window {
  MSStream?: unknown;
}

const getIsIOS = () => {
  if (typeof window === "undefined") return false;
  return (
    /iPad|iPhone|iPod/.test(navigator.userAgent) &&
    !(window as WindowWithMSStream).MSStream
  );
};

/**
 * True on iOS. The user agent never changes for the life of the page, so
 * there is nothing to subscribe to -- this reads the value on the client
 * without an effect, and reports `false` to the server renderer.
 */
export function useIsIOS() {
  return useSyncExternalStore(noopSubscribe, getIsIOS, serverFalse);
}
