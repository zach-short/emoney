import {
  useCallback,
  useEffect,
  useEffectEvent,
  useRef,
  useState,
} from "react";
import { ApiResponse } from "@/lib/utils/api.service";

// D4's retry shape (room-state-sync DESIGN.md D4; PLAN.md dial 3.2): once a
// refetch fails after the data has loaded, try again this long after each
// failure, then stop and wait for the next `refetch()`. The number of retries
// is this list's length.
const REFETCH_RETRY_DELAYS_MS = [1000, 2000, 4000] as const;

// A request that has not settled after this long counts as failed
// (room-state-sync PLAN.md BD-8). Single-flight means a request that never
// settles would hold every later trigger behind it, the socket's reconnect
// resync included, so this bounds how long one stuck GET can freeze a read.
// Twenty times the measured `/players` round trip (0.5 s, PLAN.md §0.21), and
// clear of DevTools' Slow 3G profile for a payload this size.
const REQUEST_TIMEOUT_MS = 10_000;

// The request state of one effect run. A run lasts from the effect starting
// (mount, or a change of `enabled` or `dependencies`) to its cleanup, and
// every request belongs to exactly one. A request that outlives its run sees
// `disposed` and drops its response, so nothing it carries -- data, error,
// loading, a retry, a follow-up -- reaches state after unmount or lands on top
// of a newer dependency's data.
interface FetchRun {
  disposed: boolean;
  inFlight: boolean;
  // A `refetch()` arrived while a request was in flight. Exactly one
  // follow-up fires when that request settles, and it is the one that sees
  // every commit the callers were told of.
  dirty: boolean;
  // This run has applied a success, so a later failure keeps that data (D4).
  succeeded: boolean;
  // A failure after a success has been reported to `onRefetchError`, and no
  // success has ended it yet: one report per failure cycle.
  failing: boolean;
  retriesUsed: number;
  retryTimer: ReturnType<typeof setTimeout> | null;
  // Armed while a request is in flight; fires REQUEST_TIMEOUT_MS later.
  requestTimer: ReturnType<typeof setTimeout> | null;
  trigger: () => void;
}

export function usePublicFetch<T, P = any>(
  resourceFn: (...args: P[]) => Promise<ApiResponse<T>>,
  options: {
    resourceParams?: P[];
    dependencies?: any[];
    enabled?: boolean;
    // Called once per failure cycle: the first failed refetch after a
    // success, and not again until a success ends it. Those failures keep
    // the last good `data` and do not set `error` (D4), so this is how the
    // caller hears of them. A failure before the first success sets `error`
    // instead, as it always has, and does not call this.
    onRefetchError?: () => void;
  } = {},
) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<any>(null);
  const [loading, setLoading] = useState<boolean>(true);

  const {
    resourceParams = [],
    dependencies = [],
    enabled = true,
    onRefetchError,
  } = options;

  // Every request this hook makes takes the next number, across runs, and a
  // response is applied only if its number is above the last one applied.
  // Within a run the single-flight rule below already delivers responses in
  // order; this is the stated guard that holds even if that rule is loosened.
  const requestSeqRef = useRef(0);
  const appliedSeqRef = useRef(0);
  const runRef = useRef<FetchRun | null>(null);

  const notifyRefetchError = useEffectEvent(() => {
    onRefetchError?.();
  });

  // `resourceFn` and `resourceParams` are re-created by the caller on every
  // render, so they must not drive the effect. Reading them through an effect
  // event keeps each fetch on the latest values while the effect itself only
  // re-runs when `enabled` or the caller's own `dependencies` change.
  const getRequest = useEffectEvent(() => ({
    fn: resourceFn,
    params: resourceParams,
  }));

  // `dependencies` is a fresh array literal each render, so compare by value.
  // Callers pass primitives (ids, room codes), which serialize stably.
  const dependencyKey = JSON.stringify(dependencies);

  // The mount fetch and every `refetch()` go through this run's one `request`,
  // so the two paths cannot race: at most one request is in flight per run,
  // and a trigger that arrives mid-flight is folded into one trailing
  // follow-up rather than a second concurrent GET.
  useEffect(() => {
    if (!enabled) return;

    const run: FetchRun = {
      disposed: false,
      inFlight: false,
      dirty: false,
      succeeded: false,
      failing: false,
      retriesUsed: 0,
      retryTimer: null,
      requestTimer: null,
      trigger: () => {},
    };

    const request = async () => {
      run.inFlight = true;
      const seq = ++requestSeqRef.current;
      setLoading(true);

      const { fn, params } = getRequest();

      // Past REQUEST_TIMEOUT_MS the request settles here as a failure and
      // takes the failure path below like any other: `inFlight` clears, then
      // the retry, and D4's toast once the room has loaded. The request itself
      // is not aborted, because `resourceFn` takes no signal; its late
      // response loses the race and is never read. The error carries no
      // `message`, because `Fallback` prints one (`fallback.tsx:44-46`) and a
      // new string there would be new user-facing copy.
      const timedOut = new Promise<ApiResponse<T>>((resolve) => {
        run.requestTimer = setTimeout(() => {
          run.requestTimer = null;
          resolve({ success: false, error: { status: 408, timedOut: true } });
        }, REQUEST_TIMEOUT_MS);
      });

      let response: ApiResponse<T>;
      try {
        response = await Promise.race([fn(...params), timedOut]);
      } catch (error) {
        response = { success: false, error };
      }

      if (run.requestTimer !== null) {
        clearTimeout(run.requestTimer);
        run.requestTimer = null;
      }

      // Unmounted, or the caller's dependencies moved on while this was on
      // the wire: the response is for a resource this run no longer serves.
      if (run.disposed) return;

      run.inFlight = false;

      if (seq > appliedSeqRef.current) {
        if (response.success) {
          appliedSeqRef.current = seq;
          run.succeeded = true;
          run.failing = false;
          run.retriesUsed = 0;
          setData(response.data ?? null);
          setError(null);
        } else if (!run.succeeded) {
          // Before the first success a failure is what it always was: no
          // data, and an `error` the caller's initial-load state reads.
          appliedSeqRef.current = seq;
          setData(null);
          setError(response.error);
        } else {
          // After a success a failure keeps the last good data and leaves
          // `error` alone (D4): report it once per cycle, and retry.
          if (!run.failing) {
            run.failing = true;
            notifyRefetchError();
          }

          // A pending follow-up is already the next attempt, and a fresher
          // one than a retry, so a dirty run takes it and schedules nothing.
          if (
            !run.dirty &&
            run.retriesUsed < REFETCH_RETRY_DELAYS_MS.length
          ) {
            const delay = REFETCH_RETRY_DELAYS_MS[run.retriesUsed];
            run.retriesUsed += 1;
            run.retryTimer = setTimeout(() => {
              run.retryTimer = null;
              request();
            }, delay);
          }
        }
      }

      if (run.dirty) {
        run.dirty = false;
        request();
      } else if (run.retryTimer === null) {
        setLoading(false);
      }
    };

    run.trigger = () => {
      // A trigger is fresher news than a pending retry: fetch now instead,
      // and give this trigger its own full set of retries.
      if (run.retryTimer !== null) {
        clearTimeout(run.retryTimer);
        run.retryTimer = null;
      }
      run.retriesUsed = 0;

      if (run.inFlight) {
        run.dirty = true;
        return;
      }

      request();
    };

    runRef.current = run;
    request();

    return () => {
      run.disposed = true;
      if (run.retryTimer !== null) {
        clearTimeout(run.retryTimer);
        run.retryTimer = null;
      }
      if (run.requestTimer !== null) {
        clearTimeout(run.requestTimer);
        run.requestTimer = null;
      }
      if (runRef.current === run) {
        runRef.current = null;
      }
    };
  }, [enabled, dependencyKey]);

  // Stable across renders. It reaches the current run's `request`, so it
  // coalesces with the mount fetch and with itself. While the hook is
  // disabled there is no run and it does nothing: there are no params this
  // hook has been told are good to fetch with.
  const refetch = useCallback(() => {
    runRef.current?.trigger();
  }, []);

  // Nothing is in flight while disabled, so report that during render rather
  // than writing `loading` from an effect.
  return { data, error, loading: enabled ? loading : false, refetch };
}

interface PublicActionOptions<T = any> {
  onSuccess?: (data: T) => void | Promise<void>;
  onError?: (error: any) => void | Promise<void>;
  resetOnSuccess?: boolean;
}

export function usePublicAction<T, P extends any[]>(
  resourceFn: (...args: P) => Promise<ApiResponse<T>>,
  options: PublicActionOptions<T> = {},
) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<any>(null);
  const [data, setData] = useState<T | null>(null);
  const { onSuccess, onError, resetOnSuccess = true } = options;

  const execute = async (...params: P) => {
    setLoading(true);

    if (resetOnSuccess) {
      setError(null);
      setData(null);
    }

    try {
      const response = await resourceFn(...params);

      if (response.success) {
        setData(response.data ?? null);
        if (onSuccess && response.data !== undefined) {
          await onSuccess(response.data);
        }
        return response.data ?? null;
      } else {
        setError(response.error);
        if (onError) {
          onError(response.error);
        }
        return null;
      }
    } catch (error) {
      setError(error);
      if (onError) {
        onError(error);
      }
      return null;
    } finally {
      setLoading(false);
    }
  };

  return { execute, loading, error, data };
}
