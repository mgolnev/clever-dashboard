import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api } from "../api";
import type { GoalReport } from "../types";
import {
  allocationTargets,
  buildAllocation,
  buildGoalProgress,
} from "../utils/planCalc";

export function useGoalPlan(year: number, month: number) {
  const [report, setReport] = useState<GoalReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const requestRef = useRef<AbortController | null>(null);

  const load = useCallback(async () => {
    requestRef.current?.abort();
    const controller = new AbortController();
    requestRef.current = controller;
    setLoading(true);
    setError(null);
    setReport(null);
    try {
      const next = await api.goal(year, month, controller.signal);
      if (!controller.signal.aborted) setReport(next);
    } catch (caught) {
      if (caught instanceof DOMException && caught.name === "AbortError") return;
      setError(caught instanceof Error ? caught.message : "Ошибка загрузки цели");
    } finally {
      if (requestRef.current === controller) {
        requestRef.current = null;
        setLoading(false);
      }
    }
  }, [year, month]);

  useEffect(() => {
    void load();
    return () => {
      const controller = requestRef.current;
      requestRef.current = null;
      controller?.abort();
    };
  }, [load]);

  const allocation = useMemo(() => {
    if (!report) return null;
    return buildAllocation(report.history, report.current);
  }, [report]);

  const progress = useMemo(() => {
    if (!report || !allocation) return null;
    return buildGoalProgress({
      year,
      month,
      target: report.plan.months[month - 1]?.targets.all ?? 0,
      bounds: report.bounds,
      allocation,
      current: report.current,
      historical: report.history,
    });
  }, [report, allocation, year, month]);

  const saveTarget = useCallback(async (total: number) => {
    if (!allocation) return;
    setSaving(true);
    setError(null);
    try {
      const targets = allocationTargets(total, allocation);
      const plan = await api.putPlan(year, [
        { month, channel: "all", netTarget: targets.all },
        { month, channel: "site", netTarget: targets.site },
        { month, channel: "app", netTarget: targets.app },
      ]);
      setReport((current) => current ? { ...current, plan } : current);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Ошибка сохранения цели");
    } finally {
      setSaving(false);
    }
  }, [allocation, year, month]);

  return {
    plan: report?.plan ?? null,
    bounds: report?.bounds ?? null,
    analyticsStatus: report?.analyticsStatus ?? null,
    allocation,
    progress,
    loading,
    saving,
    error,
    saveTarget,
    reload: load,
  };
}
