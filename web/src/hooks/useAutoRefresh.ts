/**
 * Auto Refresh Hook
 *
 * Custom hook to periodically update data
 * Polling solution until WebSocket implementation
 */

import { useEffect, useRef } from 'react';
import { config } from '../config/env';

interface UseAutoRefreshOptions {
  enabled?: boolean;
  interval?: number;
  onError?: (error: Error) => void;
}

/**
 * Auto refresh hook
 *
 * @param fetchFn - Data fetching function
 * @param options - Options
 * @returns None
 *
 * @example
 * ```typescript
 * const fetchSessions = useSessionStore((state) => state.fetchSessions);
 * useAutoRefresh(fetchSessions, {
 *   enabled: true,
 *   interval: 5000,
 * });
 * ```
 */
export function useAutoRefresh(
  fetchFn: () => Promise<void>,
  options: UseAutoRefreshOptions = {}
) {
  const {
    enabled = true,
    interval = config.pollingInterval,
    onError,
  } = options;

  const savedFetchFn = useRef(fetchFn);
  const savedOnError = useRef(onError);

  // Keep function references up to date
  useEffect(() => {
    savedFetchFn.current = fetchFn;
  }, [fetchFn]);

  useEffect(() => {
    savedOnError.current = onError;
  }, [onError]);

  useEffect(() => {
    // Skip if disabled
    if (!enabled) {
      return;
    }

    // Execute fetch function
    const executeFetch = async () => {
      try {
        await savedFetchFn.current();
      } catch (error) {
        if (savedOnError.current && error instanceof Error) {
          savedOnError.current(error);
        } else {
          console.error('Auto refresh error:', error);
        }
      }
    };

    executeFetch();

    // Set up periodic execution
    const timer = setInterval(executeFetch, interval);

    return () => clearInterval(timer);
  }, [enabled, interval]);
}

/**
 * Manage multiple auto refreshes
 *
 * @param refreshers - Refresh settings array
 *
 * @example
 * ```typescript
 * useMultiAutoRefresh([
 *   { fetchFn: fetchSessions, interval: 5000 },
 *   { fetchFn: fetchListeners, interval: 10000 },
 * ]);
 * ```
 */
export function useMultiAutoRefresh(
  refreshers: Array<{
    fetchFn: () => Promise<void>;
    enabled?: boolean;
    interval?: number;
    onError?: (error: Error) => void;
  }>
) {
  for (const refresher of refreshers) {
    useAutoRefresh(refresher.fetchFn, {
      enabled: refresher.enabled,
      interval: refresher.interval,
      onError: refresher.onError,
    });
  }
}
