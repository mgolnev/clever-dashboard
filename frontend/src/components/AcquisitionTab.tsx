import { useMemo, useState } from "react";
import type {
  AcquisitionChannel,
  AcquisitionDay,
  AcquisitionReport,
  AnalyticsSourceStatus,
  AnalyticsStatus,
} from "../types";
import {
  acquisitionBucketLabel,
  aggregateAcquisitionDays,
  type AcquisitionGranularity,
} from "../utils/acquisition";
import { delta, num, pct2 } from "../utils/format";
import DeltaBadge from "./DeltaBadge";

interface Props {
  report: AcquisitionReport;
  status: AnalyticsStatus | null;
  showCompare?: boolean;
}

const SOURCE_NAMES = {
  metrika: "Яндекс Метрика",
  appmetrica: "AppMetrica",
};

function dateTime(value?: string): string {
  if (!value) return "—";
  const normalized = value.includes("T") ? value : value.replace(" ", "T");
  const d = new Date(normalized);
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString("ru-RU");
}

function SourceState({ source, enabled }: { source: AnalyticsSourceStatus; enabled: boolean }) {
  const healthy = source.configured && source.status === "success";
  const color = healthy
    ? "bg-emerald-50 text-emerald-700 ring-emerald-200"
    : source.status === "error"
      ? "bg-rose-50 text-rose-700 ring-rose-200"
      : "bg-amber-50 text-amber-800 ring-amber-200";
  let summary = "нужны ID и OAuth-токен";
  if (!enabled) summary = "автозагрузка выключена";
  else if (source.status === "running") summary = "синхронизация…";
  else if (healthy) summary = `данные по ${source.lastDataDay || "—"}`;
  else if (source.status === "error") summary = "ошибка синхронизации";

  return (
    <div className={`rounded-lg px-3 py-2 ring-1 ${color}`}>
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
        <span className="text-sm font-semibold">{SOURCE_NAMES[source.source]}</span>
        <span className="text-xs">{summary}</span>
      </div>
      {source.finishedAt && (
        <p className="mt-0.5 text-[11px] opacity-75">Обновлено: {dateTime(source.finishedAt)}</p>
      )}
      {source.error && <p className="mt-1 max-w-2xl text-xs">{source.error}</p>}
    </div>
  );
}

function Metric({
  label,
  value,
  current,
  previous,
  showCompare,
  prominent,
}: {
  label: string;
  value: string;
  current: number;
  previous: number;
  showCompare?: boolean;
  prominent?: boolean;
}) {
  return (
    <div>
      <p className="text-[11px] font-medium uppercase tracking-wide text-slate-400">{label}</p>
      <div className="mt-1 flex flex-wrap items-center gap-2">
        <span className={`${prominent ? "text-3xl" : "text-xl"} font-semibold tabular-nums text-ink`}>{value}</span>
        {showCompare && (
          <DeltaBadge d={delta(current, previous)} mode="pct" />
        )}
      </div>
    </div>
  );
}

function ChannelCard({
  current,
  previous,
  showCompare,
}: {
  current: AcquisitionChannel;
  previous?: AcquisitionChannel;
  showCompare?: boolean;
}) {
  const prev = previous ?? current;
  const accent = current.channel === "site" ? "bg-sky-500" : current.channel === "app" ? "bg-violet-500" : "bg-indigo-600";
  const denominator = current.channel === "site" ? "визитов" : current.channel === "app" ? "сессий" : "визитов + сессий";
  return (
    <section className="relative overflow-hidden rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
      <span className={`absolute inset-x-0 top-0 h-1 ${accent}`} />
      <div className="flex items-baseline justify-between gap-3">
        <h3 className="text-base font-semibold text-ink">{current.label}</h3>
        <span className="text-xs text-slate-400">{denominator}</span>
      </div>
      <div className="mt-5">
        <Metric label="Трафик" value={num(current.sessions)} current={current.sessions} previous={prev.sessions} showCompare={showCompare} prominent />
      </div>
      <div className="mt-5 divide-y divide-slate-100 border-t border-slate-100">
        <div className="grid grid-cols-2 gap-5 py-4">
          <Metric label="Заказы" value={num(current.orders)} current={current.orders} previous={prev.orders} showCompare={showCompare} />
          <Metric label="CR в заказ" value={current.sessions > 0 ? pct2(current.orderCr) : "—"} current={current.orderCr} previous={prev.orderCr} showCompare={showCompare && current.sessions > 0} />
        </div>
        <div className="grid grid-cols-2 gap-5 pt-4">
          <Metric label="Оплачено" value={num(current.paidOrders)} current={current.paidOrders} previous={prev.paidOrders} showCompare={showCompare} />
          <Metric label="CR в оплату" value={current.sessions > 0 ? pct2(current.paidCr) : "—"} current={current.paidCr} previous={prev.paidCr} showCompare={showCompare && current.sessions > 0} />
        </div>
      </div>
    </section>
  );
}

type DynamicsChannel = AcquisitionChannel["channel"];

interface DynamicsPoint {
  day: string;
  traffic: number;
  orderCr: number;
  paidCr: number;
  orders: number;
  paidOrders: number;
}

const DYNAMICS_CHANNELS: Array<{ channel: DynamicsChannel; label: string; accent: string }> = [
  { channel: "all", label: "Итого", accent: "#4f46e5" },
  { channel: "site", label: "Сайт", accent: "#0ea5e9" },
  { channel: "app", label: "Приложение", accent: "#8b5cf6" },
];

const GRANULARITIES: Array<{ key: AcquisitionGranularity; label: string }> = [
  { key: "day", label: "Дни" },
  { key: "week", label: "Недели" },
  { key: "month", label: "Месяцы" },
];

function dynamicsPoint(point: AcquisitionDay, channel: DynamicsChannel): DynamicsPoint {
  const site = channel === "site" || channel === "all";
  const app = channel === "app" || channel === "all";
  const traffic = (site ? point.siteSessions : 0) + (app ? point.appSessions : 0);
  const orders = (site ? point.siteOrders : 0) + (app ? point.appOrders : 0);
  const paidOrders = (site ? point.sitePaidOrders : 0) + (app ? point.appPaidOrders : 0);
  return {
    day: point.day,
    traffic,
    orders,
    paidOrders,
    orderCr: traffic > 0 ? (orders / traffic) * 100 : 0,
    paidCr: traffic > 0 ? (paidOrders / traffic) * 100 : 0,
  };
}

function DailyChart({ points, channel, accent, granularity }: { points: AcquisitionDay[]; channel: DynamicsChannel; accent: string; granularity: AcquisitionGranularity }) {
  const values = points.map((point) => dynamicsPoint(point, channel));
  const maxTraffic = Math.max(1, ...values.map((point) => point.traffic));
  const maxCr = Math.max(0.1, ...values.flatMap((point) => [point.orderCr, point.paidCr])) * 1.15;
  if (!values.some((point) => point.traffic)) {
    return <p className="py-8 text-sm text-slate-400">Трафик за выбранный период пока не загружен — конверсию нельзя рассчитать.</p>;
  }

  const width = Math.max(1000, values.length * 80);
  const height = 280;
  const left = 56;
  const right = 54;
  const top = 18;
  const bottom = 42;
  const chartHeight = height - top - bottom;
  const chartWidth = width - left - right;
  const step = chartWidth / values.length;
  const barWidth = Math.min(42, step * 0.48);
  const x = (index: number) => left + step * (index + 0.5);
  const trafficY = (value: number) => top + chartHeight - (value / maxTraffic) * chartHeight;
  const crY = (value: number) => top + chartHeight - (value / maxCr) * chartHeight;
  const orderLine = values.map((point, index) => `${x(index)},${crY(point.orderCr)}`).join(" ");
  const paidLine = values.map((point, index) => `${x(index)},${crY(point.paidCr)}`).join(" ");

  return (
    <div className="overflow-x-auto pb-1">
      <div style={{ minWidth: width }}>
        <svg className="h-[280px] w-full" viewBox={`0 0 ${width} ${height}`} role="img" aria-label="Динамика трафика и конверсии">
          {[0, 0.5, 1].map((tick) => {
            const y = top + chartHeight * (1 - tick);
            return (
              <g key={tick}>
                <line x1={left} x2={width - right} y1={y} y2={y} stroke="#e2e8f0" strokeDasharray="3 5" />
                <text x={left - 8} y={y + 4} textAnchor="end" fontSize="10" fill="#94a3b8">{num(Math.round(maxTraffic * tick))}</text>
                <text x={width - right + 8} y={y + 4} fontSize="10" fill="#94a3b8">{pct2(maxCr * tick)}</text>
              </g>
            );
          })}

          {values.map((point, index) => {
            const y = trafficY(point.traffic);
            return (
              <g key={point.day}>
                <rect x={x(index) - barWidth / 2} y={y} width={barWidth} height={top + chartHeight - y} rx="5" fill={accent} opacity="0.22">
                  <title>{`${acquisitionBucketLabel(point.day, granularity)}: трафик ${num(point.traffic)}`}</title>
                </rect>
                <text x={x(index)} y={height - 15} textAnchor="middle" fontSize="10" fill="#94a3b8">{acquisitionBucketLabel(point.day, granularity)}</text>
              </g>
            );
          })}

          <polyline points={orderLine} fill="none" stroke="#334155" strokeWidth="2.5" strokeLinejoin="round" strokeLinecap="round" />
          <polyline points={paidLine} fill="none" stroke="#10b981" strokeWidth="2.5" strokeLinejoin="round" strokeLinecap="round" />

          {values.map((point, index) => (
            <g key={`${point.day}-conversion`}>
              <circle cx={x(index)} cy={crY(point.orderCr)} r="4" fill="#334155" stroke="white" strokeWidth="2">
                <title>{`${acquisitionBucketLabel(point.day, granularity)}: заказов ${num(point.orders)}, CR ${pct2(point.orderCr)}`}</title>
              </circle>
              <circle cx={x(index)} cy={crY(point.paidCr)} r="4" fill="#10b981" stroke="white" strokeWidth="2">
                <title>{`${acquisitionBucketLabel(point.day, granularity)}: оплачено ${num(point.paidOrders)}, CR ${pct2(point.paidCr)}`}</title>
              </circle>
            </g>
          ))}
        </svg>
      </div>
    </div>
  );
}

function DynamicsSection({ points, channel, label, accent, granularity }: { points: AcquisitionDay[]; channel: DynamicsChannel; label: string; accent: string; granularity: AcquisitionGranularity }) {
  return (
    <section className="relative overflow-hidden rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
      <span className="absolute inset-y-0 left-0 w-1" style={{ backgroundColor: accent }} />
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h3 className="font-semibold text-ink">{label}</h3>
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-slate-500">
          <span className="inline-flex items-center gap-1.5"><span className="h-3 w-3 rounded-sm" style={{ backgroundColor: accent, opacity: 0.3 }} />Трафик</span>
          <span className="inline-flex items-center gap-1.5"><span className="h-0.5 w-4 bg-slate-700" />CR оформленных</span>
          <span className="inline-flex items-center gap-1.5"><span className="h-0.5 w-4 bg-emerald-500" />CR оплаченных</span>
        </div>
      </div>
      <DailyChart points={points} channel={channel} accent={accent} granularity={granularity} />
    </section>
  );
}

export default function AcquisitionTab({ report, status, showCompare = true }: Props) {
  const [granularity, setGranularity] = useState<AcquisitionGranularity>("day");
  const dynamicsPoints = useMemo(
    () => aggregateAcquisitionDays(report.current.daily, granularity),
    [report.current.daily, granularity]
  );
  const prevByChannel = new Map(report.prev.channels.map((c) => [c.channel, c]));
  return (
    <div className="space-y-4">
      <section className="rounded-xl bg-white p-4 shadow-sm ring-1 ring-slate-200">
        <div className="mb-3 flex flex-wrap items-start justify-between gap-2">
          <div>
            <h2 className="font-semibold text-ink">Источники трафика</h2>
            <p className="mt-0.5 text-xs text-slate-500">Храним дневные агрегаты без идентификаторов пользователей.</p>
          </div>
          {report.current.sampled && <span className="rounded-full bg-amber-50 px-2.5 py-1 text-xs font-medium text-amber-800">Яндекс применил семплирование</span>}
        </div>
        <div className="grid gap-2 md:grid-cols-2">
          {status?.sources.map((source) => <SourceState key={source.source} source={source} enabled={status.enabled} />)}
        </div>
        {!status?.enabled && (
          <p className="mt-3 text-xs text-slate-500">Добавьте секреты backend и включите <code>ANALYTICS_SYNC_ENABLED=true</code>. После перезапуска первая историческая загрузка начнётся автоматически.</p>
        )}
      </section>

      {!report.current.hasTraffic && (
        <div className="rounded-xl bg-indigo-50 p-4 text-sm text-indigo-900 ring-1 ring-indigo-200">
          Трафик за выбранный период ещё не загружен. Заказы уже учитываются, а CR появится после первой синхронизации.
        </div>
      )}

      <div className="grid gap-4 lg:grid-cols-3">
        {report.current.channels.map((channel) => (
          <ChannelCard key={channel.channel} current={channel} previous={prevByChannel.get(channel.channel)} showCompare={showCompare} />
        ))}
      </div>

      <div className="flex flex-wrap items-end justify-between gap-3 pt-2">
        <div>
          <h2 className="font-semibold text-ink">Динамика трафика и конверсии</h2>
          <p className="mt-0.5 text-xs text-slate-500">Столбцы — трафик, линии — конверсия в оформленный и оплаченный заказ. Левая шкала — трафик, правая — CR.</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs font-medium text-slate-400">Группировка:</span>
          <div className="flex flex-wrap gap-1.5">
            {GRANULARITIES.map((item) => (
              <button
                key={item.key}
                type="button"
                onClick={() => setGranularity(item.key)}
                className={`rounded-lg px-3 py-1 text-xs transition ${
                  item.key === granularity
                    ? "bg-brand text-white"
                    : "border border-slate-300 text-slate-600 hover:border-brand hover:text-brand"
                }`}
              >
                {item.label}
              </button>
            ))}
          </div>
        </div>
      </div>
      <div className="space-y-4">
        {DYNAMICS_CHANNELS.map((item) => (
          <DynamicsSection key={item.channel} points={dynamicsPoints} granularity={granularity} {...item} />
        ))}
      </div>
    </div>
  );
}
