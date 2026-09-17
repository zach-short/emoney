import { useCallback, useEffect, useEffectEvent, useState } from "react";
import { ApiResponse } from "@/lib/utils/api.service";

export function usePublicFetch<T, P = any>(
  resourceFn: (...args: P[]) => Promise<ApiResponse<T>>,
  options: {
    resourceParams?: P[];
    dependencies?: any[];
    enabled?: boolean;
  } = {},
) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<any>(null);
  const [loading, setLoading] = useState<boolean>(true);

  const { resourceParams = [], dependencies = [], enabled = true } = options;

  const applyResponse = useCallback((response: ApiResponse<T>) => {
    if (response.success) {
      setData(response.data ?? null);
      setError(null);
    } else {
      setData(null);
      setError(response.error);
    }
  }, []);

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

  useEffect(() => {
    if (!enabled) return;

    let isMounted = true;

    const fetchData = async () => {
      setLoading(true);

      const { fn, params } = getRequest();

      try {
        const response = await fn(...params);

        if (isMounted) {
          applyResponse(response);
        }
      } catch (error) {
        if (isMounted) {
          setData(null);
          setError(error);
        }
      } finally {
        if (isMounted) {
          setLoading(false);
        }
      }
    };

    fetchData();

    return () => {
      isMounted = false;
    };
  }, [enabled, dependencyKey, applyResponse]);

  const refetch = async () => {
    setLoading(true);
    try {
      applyResponse(await resourceFn(...resourceParams));
    } catch (error) {
      setData(null);
      setError(error);
    } finally {
      setLoading(false);
    }
  };

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
