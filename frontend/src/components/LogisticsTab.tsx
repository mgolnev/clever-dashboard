import type { LogisticsReport, LogisticsServiceRow, LogisticsSummary } from "../types";
import { delta, num, numAbs, pct, ppAbs, rub, rubAbs } from "../utils/format";
import DeltaBadge from "./DeltaBadge";

interface Props {
  report: LogisticsReport;
  showCompare?: boolean;
}

interface EconomyKpi {
  label: string;
  hint: string;
  pick: (s: LogisticsSummary) => number;
  fmt: (n: number) => string;
  fmtAbs: (n: number) => string;
}

const ECONOMY_KPIS: EconomyKpi[] = [
  { label: "Заказы", hint: "гросс", pick: (s) => s.orders, fmt: num, fmtAbs: numAbs },
  { label: "Выручка", hint: "не отменённые", pick: (s) => s.revenue, fmt: rub, fmtAbs: rubAbs },
  { label: "Оплата", hint: "от оформленных", pick: (s) => s.paidRate, fmt: pct, fmtAbs: ppAbs },
  { label: "Бесплатно", hint: "доля заказов", pick: (s) => s.freeDeliveryRate, fmt: pct, fmtAbs: ppAbs },
];

function dateLabel(date?: string): string {
  if (!date) return "—";
  return new Date(`${date}T00:00:00`).toLocaleDateString("ru-RU", {
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}

function days(n: number): string {
  return `${n.toLocaleString("ru-RU", { maximumFractionDigits: 1 })} дн.`;
}

function daysAbs(n: number): string {
  const sign = n > 0 ? "+" : "";
  return `${sign}${n.toLocaleString("ru-RU", { maximumFractionDigits: 1 })} дн.`;
}

function combineSummary(a: LogisticsSummary, b: LogisticsSummary): LogisticsSummary {
  const orders = a.orders + b.orders;
  const paidOrders = a.paidOrders + b.paidOrders;
  const revenue = a.revenue + b.revenue;
  const deliveryTotal = a.deliveryTotal + b.deliveryTotal;
  const freeOrders = a.freeOrders + b.freeOrders;
  const processedOrders = a.processedOrders + b.processedOrders;
  const shippedOrders = a.shippedOrders + b.shippedOrders;
  const pendingShipment = a.pendingShipment + b.pendingShipment;
  const pendingAgeTotal = a.avgPendingAgeDays * a.pendingShipment + b.avgPendingAgeDays * b.pendingShipment;
  return {
    orders,
    paidOrders,
    revenue,
    deliveryTotal,
    freeOrders,
    processedOrders,
    shippedOrders,
    pendingShipment,
    pending0To1: a.pending0To1 + b.pending0To1,
    pending2To3: a.pending2To3 + b.pending2To3,
    pending4Plus: a.pending4Plus + b.pending4Plus,
    paidRate: orders ? (paidOrders / orders) * 100 : 0,
    avgDelivery: orders ? Math.round(deliveryTotal / orders) : 0,
    freeDeliveryRate: orders ? (freeOrders / orders) * 100 : 0,
    shipmentRate: processedOrders ? (shippedOrders / processedOrders) * 100 : 0,
    avgPendingAgeDays: pendingShipment ? pendingAgeTotal / pendingShipment : 0,
  };
}

function PrimaryMetric({
  label,
  value,
  hint,
  current,
  previous,
  showCompare,
  invert,
  fmtAbs = numAbs,
  deltaMode = "both",
  tone = "default",
}: {
  label: string;
  value: string;
  hint: string;
  current: number;
  previous: number;
  showCompare: boolean;
  invert?: boolean;
  fmtAbs?: (n: number) => string;
  deltaMode?: "both" | "pct" | "abs";
  tone?: "default" | "risk";
}) {
  return (
    <div className="min-w-0 px-4 py-4 first:pl-0 last:pr-0 sm:px-5">
      <div className="text-[11px] font-semibold uppercase tracking-[0.08em] text-slate-500">{label}</div>
      <div className={`mt-1 text-3xl font-semibold tabular-nums tracking-tight ${tone === "risk" ? "text-rose-600" : "text-ink"}`}>
        {value}
      </div>
      <div className="mt-1 text-xs text-slate-500">{hint}</div>
      {showCompare && (
        <div className="mt-2">
          <DeltaBadge d={delta(current, previous)} invert={invert} fmtAbs={fmtAbs} mode={deltaMode} />
        </div>
      )}
    </div>
  );
}

function AgeMix({ row, compact = false }: { row: LogisticsSummary | LogisticsServiceRow; compact?: boolean }) {
  if (row.pendingShipment === 0) {
    return <span className="text-xs text-emerald-700">очереди нет</span>;
  }
  const width = (value: number) => `${(value / row.pendingShipment) * 100}%`;
  return (
    <div className={compact ? "min-w-[170px]" : "w-full"}>
      <div className="flex h-2 overflow-hidden rounded-full bg-slate-100" aria-label="Возраст неотправленных заказов">
        <span className="bg-slate-400 transition-all duration-500" style={{ width: width(row.pending0To1) }} />
        <span className="bg-amber-400 transition-all duration-500" style={{ width: width(row.pending2To3) }} />
        <span className="bg-rose-500 transition-all duration-500" style={{ width: width(row.pending4Plus) }} />
      </div>
      <div className="mt-1.5 flex flex-wrap gap-x-3 gap-y-1 text-[11px] tabular-nums text-slate-500">
        <span>0–1 д: {num(row.pending0To1)}</span>
        <span className="text-amber-700">2–3 д: {num(row.pending2To3)}</span>
        <span className="text-rose-700">4+ д: {num(row.pending4Plus)}</span>
      </div>
    </div>
  );
}

function EconomySummaryRow({
  title,
  current,
  previous,
  showCompare,
  highlight,
}: {
  title: string;
  current: LogisticsSummary;
  previous: LogisticsSummary;
  showCompare: boolean;
  highlight?: boolean;
}) {
  return (
    <tr className={highlight ? "bg-indigo-50/50" : ""}>
      <td className="py-2 pr-3 font-medium text-slate-800">{title}</td>
      {ECONOMY_KPIS.map((kpi) => (
        <td key={kpi.label} className="px-2 py-2 text-right text-sm tabular-nums text-slate-700">
          <div>{kpi.fmt(kpi.pick(current))}</div>
          {showCompare && (
            <div className="mt-0.5 flex justify-end">
              <DeltaBadge d={delta(kpi.pick(current), kpi.pick(previous))} fmtAbs={kpi.fmtAbs} />
            </div>
          )}
        </td>
      ))}
    </tr>
  );
}

export default function LogisticsTab({ report, showCompare = true }: Props) {
  const { current, prev, period, previous } = report;
  const prevServices = new Map(prev.byService.map((row) => [row.name, row]));
  const prevCities = new Map(prev.byCity.map((row) => [row.name, row]));
  const riskOrders = current.summary.pending2To3 + current.summary.pending4Plus;
  const prevRiskOrders = prev.summary.pending2To3 + prev.summary.pending4Plus;

  return (
    <div className="space-y-4">
      <section className="overflow-hidden rounded-2xl bg-white shadow-sm ring-1 ring-slate-200">
        <div className="flex flex-col gap-3 border-b border-slate-100 px-5 py-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h2 className="text-lg font-semibold tracking-tight text-ink">Скорость отправки</h2>
            <p className="mt-1 max-w-2xl text-sm text-slate-500">
              Заказы, созданные в выбранном периоде: сколько дошло до обработки, сколько уже передано в доставку и что осталось в очереди.
            </p>
          </div>
          <div className="shrink-0 text-xs text-slate-500">
            Состояние данных на <span className="font-medium text-slate-700">{dateLabel(report.dataAsOf)}</span>
          </div>
        </div>

        <div className="grid grid-cols-2 px-5 sm:grid-cols-4">
          <PrimaryMetric label="Дошли до обработки" value={num(current.summary.processedOrders)} hint="база для расчёта" current={current.summary.processedOrders} previous={prev.summary.processedOrders} showCompare={showCompare} />
          <PrimaryMetric label="Отправлены" value={num(current.summary.shippedOrders)} hint="включая дальнейшие статусы" current={current.summary.shippedOrders} previous={prev.summary.shippedOrders} showCompare={showCompare} />
          <PrimaryMetric label="Не отправлены" value={num(current.summary.pendingShipment)} hint={`${num(current.summary.processedOrders)} − ${num(current.summary.shippedOrders)}`} current={current.summary.pendingShipment} previous={prev.summary.pendingShipment} showCompare={showCompare} invert tone="risk" />
          <PrimaryMetric label="Доля отправки" value={pct(current.summary.shipmentRate)} hint="от дошедших до обработки" current={current.summary.shipmentRate} previous={prev.summary.shipmentRate} showCompare={showCompare} fmtAbs={ppAbs} deltaMode="abs" />
        </div>

        <div className="border-t border-slate-100 bg-slate-50/70 px-5 py-4">
          <div className="flex items-end justify-between gap-4">
            <div>
              <div className="text-xs font-semibold uppercase tracking-wide text-slate-500">Путь до отправки</div>
              <div className="mt-1 text-sm text-slate-700">Отправлено {num(current.summary.shippedOrders)} из {num(current.summary.processedOrders)} заказов</div>
            </div>
            <span className="text-sm font-semibold tabular-nums text-brand">{pct(current.summary.shipmentRate)}</span>
          </div>
          <div className="mt-2 h-2.5 overflow-hidden rounded-full bg-slate-200">
            <div className="h-full rounded-full bg-brand transition-all duration-500" style={{ width: `${Math.min(100, current.summary.shipmentRate)}%` }} />
          </div>
        </div>
      </section>

      <section className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_280px]">
        <div className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Возраст очереди</h2>
              <p className="mt-1 text-sm text-slate-500">Календарные дни с создания заказа до даты актуальности данных.</p>
            </div>
            <div className="text-right">
              <div className="text-2xl font-semibold tabular-nums text-rose-600">{num(riskOrders)}</div>
              <div className="text-xs text-slate-500">ждут 2 дня и дольше</div>
              {showCompare && <DeltaBadge d={delta(riskOrders, prevRiskOrders)} invert fmtAbs={numAbs} />}
            </div>
          </div>
          <div className="mt-5"><AgeMix row={current.summary} /></div>
        </div>

        <div className="rounded-xl bg-slate-900 p-5 text-white shadow-sm">
          <div className="text-xs font-semibold uppercase tracking-wide text-slate-400">Средний возраст хвоста</div>
          <div className="mt-2 text-3xl font-semibold tabular-nums">{days(current.summary.avgPendingAgeDays)}</div>
          <p className="mt-2 text-xs leading-5 text-slate-400">Показатель относится только к заказам, которые ещё не отправлены.</p>
          {showCompare && <div className="mt-3"><DeltaBadge d={delta(current.summary.avgPendingAgeDays, prev.summary.avgPendingAgeDays)} invert fmtAbs={daysAbs} /></div>}
        </div>
      </section>

      <section className="overflow-hidden rounded-xl bg-white shadow-sm ring-1 ring-slate-200">
        <div className="flex flex-col gap-2 border-b border-slate-100 px-5 py-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Отправка по службам доставки</h2>
            <p className="mt-1 text-sm text-slate-500">Сначала службы с самым большим неотправленным хвостом.</p>
          </div>
          {showCompare && <span className="text-xs text-slate-400">{period.start} — {period.end} к {previous.start} — {previous.end}</span>}
        </div>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[920px] text-sm">
            <thead>
              <tr className="border-b border-slate-200 text-left text-[11px] uppercase tracking-wide text-slate-400">
                <th className="px-5 py-3 font-medium">Служба</th>
                <th className="px-3 py-3 text-right font-medium">Обработка</th>
                <th className="px-3 py-3 text-right font-medium">Отправлено</th>
                <th className="px-3 py-3 text-right font-medium">Не отправлено</th>
                <th className="w-40 px-3 py-3 font-medium">Доля отправки</th>
                <th className="px-5 py-3 font-medium">Возраст очереди</th>
              </tr>
            </thead>
            <tbody>
              {current.byService.map((row) => {
                const previousRow = prevServices.get(row.name);
                return (
                  <tr key={row.name} className={`border-b border-slate-100 transition-colors last:border-0 hover:bg-slate-50 ${row.pending4Plus > 0 ? "bg-rose-50/30" : ""}`}>
                    <td className="max-w-[240px] px-5 py-3 font-medium text-slate-700" title={row.name}>{row.name}</td>
                    <td className="px-3 py-3 text-right tabular-nums text-slate-600">{num(row.processedOrders)}</td>
                    <td className="px-3 py-3 text-right tabular-nums text-slate-700">{num(row.shippedOrders)}</td>
                    <td className="px-3 py-3 text-right tabular-nums">
                      <div className={row.pendingShipment > 0 ? "font-semibold text-rose-600" : "text-slate-500"}>{num(row.pendingShipment)}</div>
                      {showCompare && <div className="mt-1 flex justify-end"><DeltaBadge d={delta(row.pendingShipment, previousRow?.pendingShipment ?? 0)} invert fmtAbs={numAbs} /></div>}
                    </td>
                    <td className="px-3 py-3">
                      <div className="flex items-center justify-between gap-2 text-xs tabular-nums">
                        <span className="font-medium text-slate-700">{pct(row.shipmentRate)}</span>
                        {showCompare && previousRow && <span className="text-slate-400">{ppAbs(row.shipmentRate - previousRow.shipmentRate)}</span>}
                      </div>
                      <div className="mt-1.5 h-1.5 overflow-hidden rounded-full bg-slate-100"><div className="h-full rounded-full bg-brand transition-all duration-500" style={{ width: `${Math.min(100, row.shipmentRate)}%` }} /></div>
                    </td>
                    <td className="px-5 py-3"><AgeMix row={row} compact /></td>
                  </tr>
                );
              })}
            </tbody>
            <tfoot>
              <tr className="border-t-2 border-slate-200 bg-slate-50/60 font-semibold text-slate-700">
                <td className="px-5 py-3">Итого</td>
                <td className="px-3 py-3 text-right tabular-nums">{num(current.summary.processedOrders)}</td>
                <td className="px-3 py-3 text-right tabular-nums">{num(current.summary.shippedOrders)}</td>
                <td className="px-3 py-3 text-right tabular-nums text-rose-600">{num(current.summary.pendingShipment)}</td>
                <td className="px-3 py-3 tabular-nums">{pct(current.summary.shipmentRate)}</td>
                <td className="px-5 py-3"><AgeMix row={current.summary} compact /></td>
              </tr>
            </tfoot>
          </table>
        </div>
      </section>

      <details className="rounded-xl bg-white shadow-sm ring-1 ring-slate-200">
        <summary className="cursor-pointer list-none px-5 py-4 text-sm font-semibold text-slate-700">
          Как считается показатель <span className="ml-2 font-normal text-slate-400">методика и ограничения данных</span>
        </summary>
        <div className="border-t border-slate-100 px-5 py-4 text-sm leading-6 text-slate-600">
          <p><strong className="text-slate-800">Дошли до обработки</strong> — текущие статусы «в обработке/сборке» и все последующие статусы.</p>
          <p><strong className="text-slate-800">Отправлены</strong> — «Отправлен», «Прибыл в ПВЗ», «Выполнен» и возвраты.</p>
          <p><strong className="text-slate-800">Не отправлены</strong> = дошли до обработки − отправлены. Возраст считается от создания заказа, потому что выгрузка не содержит историю переходов между статусами.</p>
        </div>
      </details>

      <section className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
        <div className="mb-4">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Стоимость и география</h2>
          <p className="mt-1 text-sm text-slate-500">Вторичный контекст: экономика доставки, пилотные города и распределение заказов.</p>
        </div>
        <div className="grid grid-cols-2 divide-x divide-y divide-slate-100 border-y border-slate-100 sm:grid-cols-4 sm:divide-y-0">
          {ECONOMY_KPIS.map((kpi) => (
            <div key={kpi.label} className="px-4 py-3 first:pl-0">
              <div className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">{kpi.label}</div>
              <div className="text-[10px] text-slate-400">{kpi.hint}</div>
              <div className="mt-2 text-lg font-semibold tabular-nums text-ink">{kpi.fmt(kpi.pick(current.summary))}</div>
              {showCompare && <DeltaBadge d={delta(kpi.pick(current.summary), kpi.pick(prev.summary))} fmtAbs={kpi.fmtAbs} />}
            </div>
          ))}
        </div>

        {current.cohorts && (
          <div className="mt-5 overflow-x-auto">
            <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Пилот vs контроль</h3>
            <table className="w-full min-w-[640px] text-sm">
              <thead><tr className="border-b border-slate-200 text-left text-xs uppercase tracking-wide text-slate-400"><th className="py-2 pr-2 font-medium">Когорта</th>{ECONOMY_KPIS.map((kpi) => <th key={kpi.label} className="px-2 py-2 text-right font-medium">{kpi.label}</th>)}</tr></thead>
              <tbody>
                <EconomySummaryRow title="Пилот" current={current.cohorts.pilot} previous={prev.cohorts?.pilot ?? current.cohorts.pilot} showCompare={showCompare} highlight />
                <EconomySummaryRow title="Контроль" current={current.cohorts.control} previous={prev.cohorts?.control ?? current.cohorts.control} showCompare={showCompare} />
              </tbody>
              <tfoot><EconomySummaryRow title="Итого" current={combineSummary(current.cohorts.pilot, current.cohorts.control)} previous={combineSummary(prev.cohorts?.pilot ?? current.cohorts.pilot, prev.cohorts?.control ?? current.cohorts.control)} showCompare={showCompare} /></tfoot>
            </table>
          </div>
        )}

        <div className="mt-5 overflow-x-auto">
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">По городам</h3>
          <table className="w-full min-w-[720px] text-sm">
            <thead><tr className="border-b border-slate-200 text-left text-xs uppercase tracking-wide text-slate-400"><th className="py-2 pr-2 font-medium">Город</th><th className="px-2 py-2 text-right font-medium">Заказы</th><th className="px-2 py-2 text-right font-medium">Доля</th><th className="px-2 py-2 text-right font-medium">Выручка</th><th className="px-2 py-2 text-right font-medium">Оплата</th><th className="px-2 py-2 text-right font-medium">Бесплатно</th></tr></thead>
            <tbody>
              {current.byCity.map((row) => {
                const previousRow = prevCities.get(row.name);
                return (
                  <tr key={row.name} className={`border-b border-slate-100 last:border-0 ${row.isPilot ? "bg-indigo-50/40" : ""}`}>
                    <td className="py-2 pr-2 text-slate-700">{row.name}{row.isPilot && <span className="ml-2 rounded-full bg-indigo-100 px-1.5 py-0.5 text-[10px] font-medium text-indigo-800">пилот</span>}</td>
                    <td className="px-2 py-2 text-right tabular-nums"><div>{num(row.orders)}</div>{showCompare && <div className="mt-0.5 flex justify-end"><DeltaBadge d={delta(row.orders, previousRow?.orders ?? 0)} fmtAbs={numAbs} /></div>}</td>
                    <td className="px-2 py-2 text-right tabular-nums text-slate-500">{pct(row.share)}</td>
                    <td className="px-2 py-2 text-right tabular-nums">{rub(row.revenue)}</td>
                    <td className="px-2 py-2 text-right tabular-nums">{pct(row.paidRate)}</td>
                    <td className="px-2 py-2 text-right tabular-nums">{pct(row.freeDeliveryRate)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
