import { useEffect, useState } from "react";
import { api, type Filters } from "../api";
import type { LogisticsDynamics, LogisticsReport, LogisticsWeekPoint } from "../types";
import { num, pct, rub } from "../utils/format";

interface Props {
  report: LogisticsReport;
  start: string;
  end: string;
  filters: Filters;
  compareStart?: string;
  compareEnd?: string;
  showCompare?: boolean;
}

type Granularity = "day" | "week" | "month";

interface SeriesMetric {
  key: string;
  label: string;
  pick: (p: LogisticsWeekPoint) => number;
  fmt: (n: number) => string;
}

const GRANULARITIES: { key: Granularity; label: string; title: string }[] = [
  { key: "day", label: "Дни", title: "по дням" },
  { key: "week", label: "Недели", title: "по неделям" },
  { key: "month", label: "Месяцы", title: "по месяцам" },
];

const SERIES_METRICS: SeriesMetric[] = [
  { key: "orders", label: "Заказы", pick: (p) => p.orders, fmt: num },
  { key: "paidOrders", label: "Оплаты", pick: (p) => p.paidOrders, fmt: num },
  { key: "paidRate", label: "Оплата %", pick: (p) => p.paidRate, fmt: pct },
  { key: "revenue", label: "Выручка", pick: (p) => p.revenue, fmt: rub },
  { key: "units", label: "Товары", pick: (p) => p.units, fmt: num },
  { key: "aov", label: "Средний чек", pick: (p) => p.aov, fmt: rub },
  { key: "asp", label: "ASP", pick: (p) => p.asp, fmt: rub },
  { key: "upt", label: "UPT", pick: (p) => p.upt, fmt: (n) => n.toFixed(2) },
  { key: "avgDelivery", label: "Ср. доставка", pick: (p) => p.avgDelivery, fmt: rub },
  { key: "freeDeliveryRate", label: "Бесплатно %", pick: (p) => p.freeDeliveryRate, fmt: pct },
];

const BREAKDOWNS: { key: string; label: string }[] = [
  { key: "none", label: "Без разреза" },
  { key: "city", label: "Города" },
  { key: "delivery", label: "Доставка" },
  { key: "payment", label: "Оплата" },
  { key: "channel", label: "Витрина" },
  { key: "coupon", label: "Промокод" },
];

const COLORS = [
  "bg-indigo-500",
  "bg-emerald-500",
  "bg-amber-500",
  "bg-rose-500",
  "bg-sky-500",
  "bg-violet-500",
  "bg-teal-500",
  "bg-fuchsia-500",
];

function metricByKey(key: string): SeriesMetric {
  return SERIES_METRICS.find((m) => m.key === key) ?? SERIES_METRICS[0];
}

function granularityTitle(g: Granularity): string {
  return GRANULARITIES.find((x) => x.key === g)?.title ?? "по неделям";
}

function formatBucketLabel(bucket: string, granularity: Granularity): string {
  switch (granularity) {
    case "day":
      return bucket.slice(5);
    case "month":
      return bucket.slice(0, 7);
    default:
      return bucket.slice(5);
  }
}

function chartMinWidth(count: number, granularity: Granularity): number | undefined {
  if (granularity === "day" && count > 14) return count * 28;
  if (granularity === "week" && count > 20) return count * 32;
  return undefined;
}

function MetricSwitcher({
  metricKey,
  onChange,
}: {
  metricKey: string;
  onChange: (k: string) => void;
}) {
  return (
    <div className="flex flex-wrap gap-1">
      {SERIES_METRICS.map((m) => (
        <button
          key={m.key}
          type="button"
          onClick={() => onChange(m.key)}
          className={`rounded-full px-2.5 py-0.5 text-xs font-medium transition ${
            m.key === metricKey
              ? "bg-indigo-500 text-white"
              : "bg-slate-100 text-slate-600 hover:bg-slate-200"
          }`}
        >
          {m.label}
        </button>
      ))}
    </div>
  );
}

function SingleChart({
  points,
  prevPoints,
  metricKey,
  granularity,
  showCompare,
}: {
  points: LogisticsWeekPoint[];
  prevPoints?: LogisticsWeekPoint[];
  metricKey: string;
  granularity: Granularity;
  showCompare?: boolean;
}) {
  const metric = metricByKey(metricKey);
  if (points.length === 0) {
    return <div className="text-sm text-slate-400">Нет данных за период</div>;
  }

  // Выравнивание по индексу слева: i-й бакет текущего периода ↔ i-й бакет сравнения
  // (длины могут отличаться — рисуем overlay только для min(len)).
  const hasOverlay = showCompare && prevPoints && prevPoints.length > 0;
  const overlayLen = hasOverlay ? Math.min(points.length, prevPoints!.length) : 0;

  const allValues = [
    ...points.map((p) => metric.pick(p)),
    ...(hasOverlay ? prevPoints!.slice(0, overlayLen).map((p) => metric.pick(p)) : []),
  ];
  const max = Math.max(1, ...allValues);
  const minW = chartMinWidth(points.length, granularity);

  return (
    <div className="space-y-3">
      {hasOverlay && (
        <div className="flex items-center gap-4 text-[11px] text-slate-500">
          <span className="flex items-center gap-1">
            <span className="inline-block h-2.5 w-5 rounded bg-indigo-400" />
            Текущий
          </span>
          <span className="flex items-center gap-1">
            <span className="inline-block h-2.5 w-5 rounded border border-slate-400 bg-slate-200/60" />
            Сравнение
          </span>
        </div>
      )}
      <div className={minW ? "overflow-x-auto pb-1" : undefined}>
        <div
          className="flex h-64 items-end gap-1.5"
          style={minW ? { minWidth: minW } : undefined}
        >
          {points.map((p, i) => {
            const v = metric.pick(p);
            const h = Math.max(4, (v / max) * 90);
            const prevP = hasOverlay && i < overlayLen ? prevPoints![i] : null;
            const pv = prevP ? metric.pick(prevP) : 0;
            const ph = prevP ? Math.max(2, (pv / max) * 90) : 0;
            return (
              <div
                key={p.week}
                className="group relative flex h-full min-w-0 flex-1 flex-col justify-end"
                style={minW ? { minWidth: 24, flex: "0 0 24px" } : undefined}
                title={
                  prevP
                    ? `${p.week}: ${metric.label} ${metric.fmt(v)} · сравн. ${metric.fmt(pv)}`
                    : `${p.week}: ${metric.label} ${metric.fmt(v)}`
                }
              >
                <span className="mb-1 w-full truncate text-center text-[10px] font-medium tabular-nums text-slate-500">
                  {metric.fmt(v)}
                </span>
                <div className="relative flex w-full flex-1 items-end justify-center">
                  {prevP && (
                    <div
                      className="absolute bottom-0 w-[70%] rounded-t border border-slate-400/60 bg-slate-300/50"
                      style={{ height: `${ph}%` }}
                    />
                  )}
                  <div
                    className="relative z-10 w-full rounded-t bg-indigo-400 transition group-hover:bg-indigo-500"
                    style={{ height: `${h}%` }}
                  />
                </div>
              </div>
            );
          })}
        </div>
      </div>
      <div className={minW ? "overflow-x-auto" : undefined}>
        <div
          className="flex gap-1.5"
          style={minW ? { minWidth: minW } : undefined}
        >
          {points.map((p) => (
            <div
              key={p.week}
              className="min-w-0 flex-1 text-center text-[9px] text-slate-400"
              style={minW ? { minWidth: 24, flex: "0 0 24px" } : undefined}
            >
              {formatBucketLabel(p.week, granularity)}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function GroupedChart({
  data,
  metricKey,
  granularity,
}: {
  data: LogisticsDynamics;
  metricKey: string;
  granularity: Granularity;
}) {
  const metric = metricByKey(metricKey);
  const { weeks, groups } = data;
  if (weeks.length === 0 || groups.length === 0) {
    return <div className="text-sm text-slate-400">Нет данных за период</div>;
  }
  const max = Math.max(
    1,
    ...groups.flatMap((g) => g.points.map((p) => metric.pick(p)))
  );
  const showValues = groups.length <= 4;
  const minW = chartMinWidth(weeks.length, granularity);
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-x-4 gap-y-1.5">
        {groups.map((g, i) => (
          <div key={g.name} className="flex items-center gap-1.5 text-xs text-slate-600">
            <span className={`h-3 w-3 shrink-0 rounded-sm ${COLORS[i % COLORS.length]}`} />
            <span className="max-w-[200px] truncate" title={g.name}>
              {g.name}
            </span>
          </div>
        ))}
      </div>
      <div className={minW ? "overflow-x-auto pb-1" : undefined}>
        <div
          className="flex h-64 items-end gap-2"
          style={minW ? { minWidth: minW } : undefined}
        >
          {weeks.map((w, wi) => (
            <div
              key={w}
              className="flex h-full min-w-0 flex-1 flex-col justify-end"
              style={minW ? { minWidth: 32, flex: "0 0 32px" } : undefined}
            >
              <div className="flex h-full items-end justify-center gap-0.5">
                {groups.map((g, gi) => {
                  const p = g.points[wi];
                  const v = p ? metric.pick(p) : 0;
                  const h = Math.max(1, (v / max) * 92);
                  return (
                    <div
                      key={g.name}
                      className="flex h-full min-w-0 flex-1 flex-col justify-end"
                      title={`${g.name} · ${w}: ${metric.label} ${metric.fmt(v)}`}
                    >
                      {showValues && v > 0 && (
                        <span className="mb-0.5 w-full truncate text-center text-[8px] font-medium leading-none tabular-nums text-slate-500">
                          {metric.fmt(v)}
                        </span>
                      )}
                      <div
                        className={`w-full rounded-t transition hover:opacity-80 ${COLORS[gi % COLORS.length]}`}
                        style={{ height: `${h}%` }}
                      />
                    </div>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      </div>
      <div className={minW ? "overflow-x-auto" : undefined}>
        <div
          className="flex gap-2"
          style={minW ? { minWidth: minW } : undefined}
        >
          {weeks.map((w) => (
            <div
              key={w}
              className="min-w-0 flex-1 text-center text-[9px] text-slate-400"
              style={minW ? { minWidth: 32, flex: "0 0 32px" } : undefined}
            >
              {formatBucketLabel(w, granularity)}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

export default function DynamicsTab({
  report,
  start,
  end,
  filters,
  compareStart,
  compareEnd,
  showCompare = true,
}: Props) {
  const { period, previous } = report;
  const [metricKey, setMetricKey] = useState("orders");
  const [granularity, setGranularity] = useState<Granularity>("week");
  const [breakdown, setBreakdown] = useState("none");
  const [series, setSeries] = useState<LogisticsWeekPoint[]>(report.current.series);
  const [prevSeries, setPrevSeries] = useState<LogisticsWeekPoint[]>(report.prev.series);
  const [seriesLoading, setSeriesLoading] = useState(false);
  const [seriesError, setSeriesError] = useState<string | null>(null);
  const [dyn, setDyn] = useState<LogisticsDynamics | null>(null);
  const [dynLoading, setDynLoading] = useState(false);
  const [dynError, setDynError] = useState<string | null>(null);

  const fKey = JSON.stringify(filters);
  const compareKey = `${compareStart ?? ""}|${compareEnd ?? ""}`;

  useEffect(() => {
    if (granularity === "week") {
      setSeries(report.current.series);
      setPrevSeries(report.prev.series);
      setSeriesError(null);
      setSeriesLoading(false);
      return;
    }
    let cancelled = false;
    setSeriesLoading(true);
    setSeriesError(null);
    api
      .logistics(start, end, filters, granularity, compareStart, compareEnd)
      .then((r) => {
        if (!cancelled) {
          setSeries(r.current.series);
          setPrevSeries(r.prev.series);
        }
      })
      .catch((e) => {
        if (!cancelled) {
          setSeries([]);
          setPrevSeries([]);
          setSeriesError(e.message);
        }
      })
      .finally(() => {
        if (!cancelled) setSeriesLoading(false);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [granularity, start, end, fKey, compareKey, report.current.series, report.prev.series]);

  useEffect(() => {
    if (breakdown === "none") {
      setDyn(null);
      setDynError(null);
      return;
    }
    let cancelled = false;
    setDynLoading(true);
    setDynError(null);
    api
      .dynamics(start, end, filters, breakdown, granularity)
      .then((d) => {
        if (!cancelled) setDyn(d);
      })
      .catch((e) => {
        if (!cancelled) {
          setDyn(null);
          setDynError(e.message);
        }
      })
      .finally(() => {
        if (!cancelled) setDynLoading(false);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [breakdown, granularity, start, end, fKey]);

  const chartLoading = breakdown === "none" ? seriesLoading : dynLoading;
  const chartError = breakdown === "none" ? seriesError : dynError;

  return (
    <div className="space-y-4">
      {showCompare && (
        <p className="text-xs text-slate-500">
          Период {period.start} — {period.end} · сравнение с {previous.start} — {previous.end}
        </p>
      )}
      <div className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-500">
          Динамика {granularityTitle(granularity)}
        </h2>
        <div className="space-y-3">
          <MetricSwitcher metricKey={metricKey} onChange={setMetricKey} />
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-xs font-medium text-slate-400">Группировка:</span>
            <div className="flex flex-wrap gap-1.5">
              {GRANULARITIES.map((g) => (
                <button
                  key={g.key}
                  type="button"
                  onClick={() => setGranularity(g.key)}
                  className={`rounded-lg px-3 py-1 text-xs transition ${
                    g.key === granularity
                      ? "bg-brand text-white"
                      : "border border-slate-300 text-slate-600 hover:border-brand hover:text-brand"
                  }`}
                >
                  {g.label}
                </button>
              ))}
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-xs font-medium text-slate-400">Разрез:</span>
            <div className="flex flex-wrap gap-1.5">
              {BREAKDOWNS.map((b) => (
                <button
                  key={b.key}
                  type="button"
                  onClick={() => setBreakdown(b.key)}
                  className={`rounded-lg px-3 py-1 text-xs transition ${
                    b.key === breakdown
                      ? "bg-brand text-white"
                      : "border border-slate-300 text-slate-600 hover:border-brand hover:text-brand"
                  }`}
                >
                  {b.label}
                </button>
              ))}
            </div>
          </div>

          {chartLoading && (
            <div className="py-8 text-center text-sm text-slate-400">Загрузка…</div>
          )}
          {!chartLoading && chartError && (
            <div className="py-8 text-center text-sm text-rose-600">{chartError}</div>
          )}
          {!chartLoading && !chartError && breakdown === "none" && (
            <SingleChart
              points={series}
              prevPoints={prevSeries}
              metricKey={metricKey}
              granularity={granularity}
              showCompare={showCompare}
            />
          )}
          {!chartLoading && !chartError && breakdown !== "none" && dyn && (
            <GroupedChart data={dyn} metricKey={metricKey} granularity={granularity} />
          )}
        </div>
      </div>
    </div>
  );
}
