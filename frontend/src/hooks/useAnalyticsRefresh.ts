import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../api";
import type { AnalyticsStatus } from "../types";

const pollIntervalMs = 1_500;
const refreshTimeoutMs = 120_000;

function wait(ms: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    const onAbort = () => {
      window.clearTimeout(timer);
      reject(new DOMException("Aborted", "AbortError"));
    };
    const timer = window.setTimeout(() => {
      signal.removeEventListener("abort", onAbort);
      resolve();
    }, ms);
    signal.addEventListener("abort", onAbort, { once: true });
  });
}

export function useAnalyticsRefresh(
  onStatus: (status: AnalyticsStatus) => void,
  onComplete: () => Promise<void>
) {
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const controllerRef = useRef<AbortController | null>(null);
  const inFlightRef = useRef(false);
  const onStatusRef = useRef(onStatus);
  const onCompleteRef = useRef(onComplete);

  useEffect(() => {
    onStatusRef.current = onStatus;
    onCompleteRef.current = onComplete;
  }, [onStatus, onComplete]);

  useEffect(() => () => controllerRef.current?.abort(), []);

  const refresh = useCallback(async () => {
    if (inFlightRef.current) return;
    inFlightRef.current = true;
    setRefreshing(true);
    setError(null);
    const controller = new AbortController();
    controllerRef.current = controller;

    try {
      await api.syncAnalytics(controller.signal);
      const deadline = Date.now() + refreshTimeoutMs;
      while (!controller.signal.aborted && Date.now() < deadline) {
        const status = await api.analyticsStatus(controller.signal);
        onStatusRef.current(status);
        if (!status.syncing) {
          await onCompleteRef.current();
          return;
        }
        await wait(pollIntervalMs, controller.signal);
      }
      throw new Error("Синхронизация занимает больше двух минут. Проверьте статус позже.");
    } catch (cause) {
      if (cause instanceof DOMException && cause.name === "AbortError") return;
      setError(cause instanceof Error ? cause.message : "Не удалось обновить аналитику");
    } finally {
      if (controllerRef.current === controller) controllerRef.current = null;
      inFlightRef.current = false;
      setRefreshing(false);
    }
  }, []);

  return { refresh, refreshing, error };
}
