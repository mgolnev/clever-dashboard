import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "../api";
import type { AcquisitionReport, AnalyticsStatus, Bounds, PlanReport, Report } from "../types";
import {
  allocationRange,
  allocationTargets,
  buildAllocation,
  buildGoalProgress,
  emptyPlanFilters,
  monthRange,
} from "../utils/planCalc";

interface GoalPlanState {
  plan: PlanReport | null;
  bounds: Bounds | null;
  metricsAll: Report | null;
  metricsSite: Report | null;
  metricsApp: Report | null;
  acquisition: AcquisitionReport | null;
  historicalMetrics: Report | null;
  historicalAcquisition: AcquisitionReport | null;
  analyticsStatus: AnalyticsStatus | null;
}

const initialState: GoalPlanState = {
  plan: null,
  bounds: null,
  metricsAll: null,
  metricsSite: null,
  metricsApp: null,
  acquisition: null,
  historicalMetrics: null,
  historicalAcquisition: null,
  analyticsStatus: null,
};

export function useGoalPlan(year: number, month: number) {
  const [state, setState] = useState<GoalPlanState>(initialState);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setState(initialState);
    try {
      const [plan, bounds, analyticsStatus] = await Promise.all([
        api.getPlan(year),
        api.bounds(),
        api.analyticsStatus(),
      ]);
      const selected = monthRange(year, month);
      const history = allocationRange(year, month, bounds);
      const [metricsAll, metricsSite, metricsApp, acquisition, historicalMetrics, historicalAcquisition] = await Promise.all([
        api.metrics(selected.start, selected.end, emptyPlanFilters),
        api.metrics(selected.start, selected.end, { ...emptyPlanFilters, channel: ["Сайт"] }),
        api.metrics(selected.start, selected.end, { ...emptyPlanFilters, channel: ["Приложение"] }),
        api.acquisition(selected.start, selected.end),
        history ? api.metrics(history.start, history.end, emptyPlanFilters) : Promise.resolve(null),
        history ? api.acquisition(history.start, history.end) : Promise.resolve(null),
      ]);
      setState({
        plan,
        bounds,
        metricsAll,
        metricsSite,
        metricsApp,
        acquisition,
        historicalMetrics,
        historicalAcquisition,
        analyticsStatus,
      });
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Ошибка загрузки цели");
    } finally {
      setLoading(false);
    }
  }, [year, month]);

  useEffect(() => {
    void load();
  }, [load]);

  const allocation = useMemo(() => {
    if (!state.metricsAll || !state.bounds) return null;
    return buildAllocation(
      state.historicalMetrics,
      state.metricsAll,
      allocationRange(year, month, state.bounds)
    );
  }, [state.metricsAll, state.historicalMetrics, state.bounds, year, month]);

  const progress = useMemo(() => {
    if (
      !state.plan || !state.bounds || !state.metricsAll || !state.metricsSite
      || !state.metricsApp || !state.acquisition || !allocation
    ) return null;
    return buildGoalProgress({
      year,
      month,
      target: state.plan.months[month - 1].targets.all,
      bounds: state.bounds,
      allocation,
      metricsAll: state.metricsAll,
      metricsSite: state.metricsSite,
      metricsApp: state.metricsApp,
      acquisition: state.acquisition,
      historicalMetrics: state.historicalMetrics,
      historicalAcquisition: state.historicalAcquisition,
    });
  }, [state, allocation, year, month]);

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
      setState((current) => ({ ...current, plan }));
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Ошибка сохранения цели");
    } finally {
      setSaving(false);
    }
  }, [allocation, year, month]);

  return {
    ...state,
    allocation,
    progress,
    loading,
    saving,
    error,
    saveTarget,
    reload: load,
  };
}
