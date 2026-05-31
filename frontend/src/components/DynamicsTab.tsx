import { useEffect, useState } from "react";
import { api, type Filters } from "../api";
import type { LogisticsDynamics, LogisticsReport, LogisticsWeekPoint } from "../types";
import { num, pct, rub } from "../utils/format";

interface Props {
  report: LogisticsReport;
  start: string;
  end: string;
  filters: Filters;
}

interface SeriesMetric {
  key: string;
  label: string;
  pick: (p: LogisticsWeekPoint) => number;
  fmt: (n: number) => string;
}

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

// палитра для значений разреза (bar + легенда).
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

// SingleChart — динамика без разреза (один ряд столбцов).
function SingleChart({ points, metricKey }: { points: LogisticsWeekPoint[]; metricKey: string }) {
  const metric = metricByKey(metricKey);
  if (points.length === 0) {
    return <div className="text-sm text-slate-400">Нет данных за период</div>;
  }
  const max = Math.max(1, ...points.map((p) => metric.pick(p)));
  return (
    <div className="space-y-3">
      <div className="flex h-64 items-end gap-1.5">
        {points.map((p) => {
          const v = metric.pick(p);
          const h = Math.max(4, (v / max) * 90);
          return (
            <div
              key={p.week}
              className="group flex h-full min-w-0 flex-1 flex-col justify-end"
              title={`${p.week}: ${metric.label} ${metric.fmt(v)}`}
            >
              <span className="mb-1 w-full truncate text-center text-[10px] font-medium tabular-nums text-slate-500">
                {metric.fmt(v)}
              </span>
              <div
                className="w-full rounded-t bg-indigo-400 transition group-hover:bg-indigo-500"
                style={{ height: `${h}%` }}
              />
            </div>
          );
        })}
      </div>
      <div className="flex gap-1.5">
        {points.map((p) => (
          <div key={p.week} className="min-w-0 flex-1 text-center text-[9px] text-slate-400">
            {p.week.slice(5)}
          </div>
        ))}
      </div>
    </div>
  );
}

// GroupedChart — динамика в разрезе: на каждой неделе группа цветных столбцов.
function GroupedChart({ data, metricKey }: { data: LogisticsDynamics; metricKey: string }) {
  const metric = metricByKey(metricKey);
  const { weeks, groups } = data;
  if (weeks.length === 0 || groups.length === 0) {
    return <div className="text-sm text-slate-400">Нет данных за период</div>;
  }
  const max = Math.max(
    1,
    ...groups.flatMap((g) => g.points.map((p) => metric.pick(p)))
  );
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
      <div className="flex h-64 items-end gap-2">
        {weeks.map((w, wi) => (
          <div key={w} className="flex h-full min-w-0 flex-1 flex-col justify-end">
            <div className="flex h-full items-end justify-center gap-0.5">
              {groups.map((g, gi) => {
                const p = g.points[wi];
                const v = p ? metric.pick(p) : 0;
                const h = Math.max(1, (v / max) * 92);
                return (
                  <div
                    key={g.name}
                    className={`min-w-0 flex-1 rounded-t transition hover:opacity-80 ${COLORS[gi % COLORS.length]}`}
                    style={{ height: `${h}%` }}
                    title={`${g.name} · ${w}: ${metric.label} ${metric.fmt(v)}`}
                  />
                );
              })}
            </div>
          </div>
        ))}
      </div>
      <div className="flex gap-2">
        {weeks.map((w) => (
          <div key={w} className="min-w-0 flex-1 text-center text-[9px] text-slate-400">
            {w.slice(5)}
          </div>
        ))}
      </div>
    </div>
  );
}

export default function DynamicsTab({ report, start, end, filters }: Props) {
  const { current, period, previous } = report;
  const [metricKey, setMetricKey] = useState("orders");
  const [breakdown, setBreakdown] = useState("none");
  const [dyn, setDyn] = useState<LogisticsDynamics | null>(null);
  const [dynLoading, setDynLoading] = useState(false);
  const [dynError, setDynError] = useState<string | null>(null);

  const fKey = JSON.stringify(filters);

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
      .dynamics(start, end, filters, breakdown)
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
  }, [breakdown, start, end, fKey]);

  return (
    <div className="space-y-4">
      <p className="text-xs text-slate-500">
        Период {period.start} — {period.end} · сравнение с {previous.start} — {previous.end}
      </p>
      <div className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-500">
          Динамика по неделям
        </h2>
        <div className="space-y-3">
          <MetricSwitcher metricKey={metricKey} onChange={setMetricKey} />
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

          {breakdown === "none" && <SingleChart points={current.series} metricKey={metricKey} />}
          {breakdown !== "none" && dynLoading && (
            <div className="py-8 text-center text-sm text-slate-400">Загрузка разреза…</div>
          )}
          {breakdown !== "none" && !dynLoading && dynError && (
            <div className="py-8 text-center text-sm text-rose-600">{dynError}</div>
          )}
          {breakdown !== "none" && !dynLoading && !dynError && dyn && (
            <GroupedChart data={dyn} metricKey={metricKey} />
          )}
        </div>
      </div>
    </div>
  );
}
