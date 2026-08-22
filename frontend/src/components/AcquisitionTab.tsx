import type {
  AcquisitionChannel,
  AcquisitionDay,
  AcquisitionReport,
  AnalyticsSourceStatus,
  AnalyticsStatus,
} from "../types";
import { delta, num, numAbs, pct, ppAbs } from "../utils/format";
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
  percentPoints,
  showCompare,
}: {
  label: string;
  value: string;
  current: number;
  previous: number;
  percentPoints?: boolean;
  showCompare?: boolean;
}) {
  return (
    <div>
      <p className="text-[11px] font-medium uppercase tracking-wide text-slate-400">{label}</p>
      <div className="mt-1 flex flex-wrap items-center gap-2">
        <span className="text-xl font-semibold tabular-nums text-ink">{value}</span>
        {showCompare && (
          <DeltaBadge
            d={delta(current, previous)}
            fmtAbs={percentPoints ? ppAbs : numAbs}
            mode={percentPoints ? "abs" : "both"}
          />
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
      <div className="mb-4 flex items-baseline justify-between gap-3">
        <h3 className="text-base font-semibold text-ink">{current.label}</h3>
        <span className="text-xs text-slate-400">CR = заказы / {denominator}</span>
      </div>
      <div className="grid grid-cols-2 gap-x-5 gap-y-5">
        <Metric label="Трафик" value={num(current.sessions)} current={current.sessions} previous={prev.sessions} showCompare={showCompare} />
        <Metric label="Заказы" value={num(current.orders)} current={current.orders} previous={prev.orders} showCompare={showCompare} />
        <Metric label="CR в заказ" value={current.sessions > 0 ? pct(current.orderCr) : "—"} current={current.orderCr} previous={prev.orderCr} percentPoints showCompare={showCompare && current.sessions > 0} />
        <Metric label="CR в оплату" value={current.sessions > 0 ? pct(current.paidCr) : "—"} current={current.paidCr} previous={prev.paidCr} percentPoints showCompare={showCompare && current.sessions > 0} />
        <Metric label="Действующие" value={num(current.netOrders)} current={current.netOrders} previous={prev.netOrders} showCompare={showCompare} />
        <Metric label="CR действующих" value={current.sessions > 0 ? pct(current.netCr) : "—"} current={current.netCr} previous={prev.netCr} percentPoints showCompare={showCompare && current.sessions > 0} />
      </div>
    </section>
  );
}

function DailyChart({ points }: { points: AcquisitionDay[] }) {
  const maxTraffic = Math.max(1, ...points.flatMap((p) => [p.siteSessions, p.appSessions]));
  if (!points.some((p) => p.siteSessions || p.appSessions || p.siteOrders || p.appOrders)) {
    return <p className="text-sm text-slate-400">За выбранный период дневных данных пока нет.</p>;
  }
  const minWidth = Math.max(640, points.length * 44);
  return (
    <div className="overflow-x-auto pb-1">
      <div className="flex h-64 items-end gap-2" style={{ minWidth }}>
        {points.map((p) => (
          <div key={p.day} className="group flex h-full min-w-0 flex-1 flex-col justify-end" title={`${p.day}: сайт ${p.siteSessions} виз. / ${p.siteOrders} зак.; приложение ${p.appSessions} сесс. / ${p.appOrders} зак.`}>
            <div className="mb-1 text-center text-[10px] tabular-nums text-slate-500">{p.siteOrders + p.appOrders} зак.</div>
            <div className="flex h-[82%] items-end justify-center gap-0.5">
              <div className="w-1/2 rounded-t bg-sky-400 transition group-hover:bg-sky-500" style={{ height: `${Math.max(p.siteSessions ? 3 : 0, (p.siteSessions / maxTraffic) * 100)}%` }} />
              <div className="w-1/2 rounded-t bg-violet-400 transition group-hover:bg-violet-500" style={{ height: `${Math.max(p.appSessions ? 3 : 0, (p.appSessions / maxTraffic) * 100)}%` }} />
            </div>
            <div className="mt-1 truncate text-center text-[9px] text-slate-400">{p.day.slice(5)}</div>
          </div>
        ))}
      </div>
    </div>
  );
}

export default function AcquisitionTab({ report, status, showCompare = true }: Props) {
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

      <section className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
          <div>
            <h2 className="font-semibold text-ink">Динамика трафика по дням</h2>
            <p className="text-xs text-slate-500">Сайт — голубой, приложение — фиолетовый; над столбцом — все заказы дня.</p>
          </div>
        </div>
        <DailyChart points={report.current.daily} />
      </section>
    </div>
  );
}
