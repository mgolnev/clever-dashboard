import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "../api";
import type { PlanReport, Report, TrafficReport } from "../types";
import { buildChannelGoals, monthRange } from "../utils/planCalc";
import PlanEditor from "./PlanEditor";
import GoalSummary from "./GoalSummary";
import WhatIf from "./WhatIf";
import PlanHelp from "./PlanHelp";

const MONTH_NAMES = [
  "Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
  "Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
];

const emptyFilters = {
  city: [] as string[],
  region: [] as string[],
  channel: [] as string[],
  payment: [] as string[],
  delivery: [] as string[],
  coupon: [] as string[],
};

export default function PlanTab() {
  const currentYear = new Date().getFullYear();
  const [year, setYear] = useState(currentYear);
  const [month, setMonth] = useState(new Date().getMonth() + 1);
  const [plan, setPlan] = useState<PlanReport | null>(null);
  const [traffic, setTraffic] = useState<TrafficReport | null>(null);
  const [metricsAll, setMetricsAll] = useState<Report | null>(null);
  const [metricsSite, setMetricsSite] = useState<Report | null>(null);
  const [metricsApp, setMetricsApp] = useState<Report | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadYearData = useCallback(async (y: number) => {
    const [p, t] = await Promise.all([api.getPlan(y), api.getTraffic(y)]);
    setPlan(p);
    setTraffic(t);
    return { p, t };
  }, []);

  const loadMetrics = useCallback(async (y: number, m: number) => {
    const { start, end } = monthRange(y, m);
    const [all, site, app] = await Promise.all([
      api.metrics(start, end, emptyFilters),
      api.metrics(start, end, { ...emptyFilters, channel: ["Сайт"] }),
      api.metrics(start, end, { ...emptyFilters, channel: ["Приложение"] }),
    ]);
    setMetricsAll(all);
    setMetricsSite(site);
    setMetricsApp(app);
  }, []);

  useEffect(() => {
    setLoading(true);
    setError(null);
    Promise.all([loadYearData(year), loadMetrics(year, month)])
      .catch((e) => setError(e instanceof Error ? e.message : "Ошибка загрузки"))
      .finally(() => setLoading(false));
  }, [year, month, loadYearData, loadMetrics]);

  const goals = useMemo(() => {
    if (!plan || !traffic || !metricsAll || !metricsSite || !metricsApp) return [];
    const planMonth = plan.months[month - 1];
    const trafficMonth = traffic.months[month - 1];
    return buildChannelGoals(
      planMonth,
      trafficMonth,
      metricsAll.current.kpi,
      metricsSite.current.kpi,
      metricsApp.current.kpi,
      year,
      month
    );
  }, [plan, traffic, metricsAll, metricsSite, metricsApp, year, month]);

  const years = Array.from({ length: 5 }, (_, i) => currentYear - 2 + i);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-4 rounded-xl bg-white p-4 shadow-sm ring-1 ring-slate-200">
        <label className="flex items-center gap-2 text-sm">
          <span className="text-slate-500">Год</span>
          <select
            value={year}
            onChange={(e) => setYear(Number(e.target.value))}
            className="rounded-md border border-slate-200 px-2 py-1.5 focus:border-brand focus:outline-none focus:ring-1 focus:ring-brand"
          >
            {years.map((y) => (
              <option key={y} value={y}>
                {y}
              </option>
            ))}
          </select>
        </label>
        <label className="flex items-center gap-2 text-sm">
          <span className="text-slate-500">Месяц для анализа</span>
          <select
            value={month}
            onChange={(e) => setMonth(Number(e.target.value))}
            className="rounded-md border border-slate-200 px-2 py-1.5 focus:border-brand focus:outline-none focus:ring-1 focus:ring-brand"
          >
            {MONTH_NAMES.map((name, i) => (
              <option key={name} value={i + 1}>
                {name}
              </option>
            ))}
          </select>
        </label>
        <div className="ml-auto">
          <PlanHelp />
        </div>
      </div>

      {error && (
        <div className="rounded-lg bg-rose-50 p-3 text-sm text-rose-700">{error}</div>
      )}

      {loading && <div className="text-sm text-slate-400">Загрузка…</div>}

      {plan && traffic && (
        <PlanEditor
          year={year}
          plan={plan}
          traffic={traffic}
          onSaved={(p, t) => {
            setPlan(p);
            setTraffic(t);
          }}
        />
      )}

      {goals.length > 0 && (
        <>
          <GoalSummary goals={goals} />
          <WhatIf goals={goals} />
        </>
      )}
    </div>
  );
}
