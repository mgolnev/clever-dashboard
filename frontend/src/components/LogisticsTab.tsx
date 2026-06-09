import type { LogisticsReport, LogisticsSummary } from "../types";
import { delta, num, pct, rub } from "../utils/format";
import DeltaBadge from "./DeltaBadge";

interface Props {
  report: LogisticsReport;
}

interface KpiDef {
  label: string;
  hint?: string;
  pick: (s: LogisticsSummary) => number;
  fmt: (n: number) => string;
  invertDelta?: boolean;
}

const KPIS: KpiDef[] = [
  { label: "Заказы", hint: "гросс", pick: (s) => s.orders, fmt: num },
  { label: "Выручка", hint: "не отменённые", pick: (s) => s.revenue, fmt: rub },
  {
    label: "Оплата",
    hint: "от оформленных",
    pick: (s) => s.paidRate,
    fmt: (n) => pct(n),
    invertDelta: false,
  },
  {
    label: "Бесплатно",
    hint: "% заказов",
    pick: (s) => s.freeDeliveryRate,
    fmt: (n) => pct(n),
  },
];

// combineSummary складывает две непересекающиеся когорты (пилот + контроль) в одну
// сводку с пересчётом производных ставок/средних.
function combineSummary(a: LogisticsSummary, b: LogisticsSummary): LogisticsSummary {
  const orders = a.orders + b.orders;
  const paidOrders = a.paidOrders + b.paidOrders;
  const revenue = a.revenue + b.revenue;
  const deliveryTotal = a.deliveryTotal + b.deliveryTotal;
  const freeOrders = a.freeOrders + b.freeOrders;
  return {
    orders,
    paidOrders,
    revenue,
    deliveryTotal,
    freeOrders,
    paidRate: orders ? (paidOrders / orders) * 100 : 0,
    avgDelivery: orders ? Math.round(deliveryTotal / orders) : 0,
    freeDeliveryRate: orders ? (freeOrders / orders) * 100 : 0,
  };
}

function SummaryRow({
  title,
  current,
  prev,
  highlight,
}: {
  title: string;
  current: LogisticsSummary;
  prev: LogisticsSummary;
  highlight?: boolean;
}) {
  return (
    <tr className={highlight ? "bg-indigo-50/50" : ""}>
      <td className="py-2 pr-3 font-medium text-slate-800">{title}</td>
      {KPIS.map((k) => (
        <td key={k.label} className="px-2 py-2 text-right tabular-nums text-sm text-slate-700">
          <div>{k.fmt(k.pick(current))}</div>
          <div className="mt-0.5 flex justify-end">
            <DeltaBadge d={delta(k.pick(current), k.pick(prev))} invert={k.invertDelta} />
          </div>
        </td>
      ))}
    </tr>
  );
}

export default function LogisticsTab({ report }: Props) {
  const { current, prev, period, previous } = report;

  const svcTotal = current.byService.reduce(
    (a, r) => ({ orders: a.orders + r.orders, paid: a.paid + r.paidOrders, free: a.free + r.freeOrders }),
    { orders: 0, paid: 0, free: 0 }
  );
  const cityTotal = current.byCity.reduce(
    (a, r) => ({
      orders: a.orders + r.orders,
      paid: a.paid + r.paidOrders,
      free: a.free + r.freeOrders,
      revenue: a.revenue + r.revenue,
    }),
    { orders: 0, paid: 0, free: 0, revenue: 0 }
  );
  const rate = (part: number, whole: number) => (whole > 0 ? pct((part / whole) * 100) : "—");

  // Сопоставление с предыдущим периодом по названию строки (для приростов).
  const prevSvc = new Map(prev.byService.map((r) => [r.name, r]));
  const prevCity = new Map(prev.byCity.map((r) => [r.name, r]));
  const svcTotalPrev = prev.byService.reduce(
    (a, r) => ({ orders: a.orders + r.orders, paid: a.paid + r.paidOrders }),
    { orders: 0, paid: 0 }
  );
  const cityTotalPrev = prev.byCity.reduce(
    (a, r) => ({
      orders: a.orders + r.orders,
      revenue: a.revenue + r.revenue,
      paid: a.paid + r.paidOrders,
    }),
    { orders: 0, revenue: 0, paid: 0 }
  );
  const prevPaidRate = (orders: number, paid: number) => (orders > 0 ? (paid / orders) * 100 : 0);

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        {KPIS.map((k) => (
          <div key={k.label} className="rounded-xl bg-white p-3 shadow-sm ring-1 ring-slate-200">
            <div className="text-xs font-semibold uppercase tracking-wide text-slate-500">{k.label}</div>
            {k.hint && <div className="text-[10px] text-slate-400">{k.hint}</div>}
            <div className="mt-2 text-lg font-bold text-ink">{k.fmt(k.pick(current.summary))}</div>
            <DeltaBadge d={delta(k.pick(current.summary), k.pick(prev.summary))} />
          </div>
        ))}
      </div>

      <p className="text-xs text-slate-500">
        Период {period.start} — {period.end} · сравнение с {previous.start} — {previous.end}
      </p>

      {current.cohorts && (
        <div className="overflow-x-auto rounded-xl bg-white p-4 shadow-sm ring-1 ring-slate-200">
          <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-500">
            Пилот vs контроль
          </h2>
          <table className="w-full min-w-[640px] text-sm">
            <thead>
              <tr className="border-b border-slate-200 text-left text-xs uppercase tracking-wide text-slate-400">
                <th className="py-2 pr-2 font-medium">Когорта</th>
                {KPIS.map((k) => (
                  <th key={k.label} className="px-2 py-2 text-right font-medium">
                    {k.label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              <SummaryRow
                title="Пилот"
                current={current.cohorts.pilot}
                prev={prev.cohorts?.pilot ?? current.cohorts.pilot}
                highlight
              />
              <SummaryRow
                title="Контроль"
                current={current.cohorts.control}
                prev={prev.cohorts?.control ?? current.cohorts.control}
              />
            </tbody>
            <tfoot>
              <tr className="border-t-2 border-slate-200">
                <td className="py-2 pr-3 font-semibold text-slate-800">Итого</td>
                {KPIS.map((k) => {
                  const curTotal = combineSummary(current.cohorts!.pilot, current.cohorts!.control);
                  const prevTotal = combineSummary(
                    prev.cohorts?.pilot ?? current.cohorts!.pilot,
                    prev.cohorts?.control ?? current.cohorts!.control
                  );
                  return (
                    <td
                      key={k.label}
                      className="px-2 py-2 text-right font-semibold tabular-nums text-sm text-slate-800"
                    >
                      <div>{k.fmt(k.pick(curTotal))}</div>
                      <div className="mt-0.5 flex justify-end">
                        <DeltaBadge d={delta(k.pick(curTotal), k.pick(prevTotal))} invert={k.invertDelta} />
                      </div>
                    </td>
                  );
                })}
              </tr>
            </tfoot>
          </table>
        </div>
      )}

      <div className="overflow-x-auto rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-500">
          Службы доставки
        </h2>
        <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-200 text-left text-xs uppercase tracking-wide text-slate-400">
                <th className="py-2 pr-2 font-medium">Служба</th>
                <th className="px-2 py-2 text-right font-medium">Заказы</th>
                <th className="px-2 py-2 text-right font-medium">Доля</th>
                <th className="px-2 py-2 text-right font-medium">Оплаты</th>
                <th className="px-2 py-2 text-right font-medium">Доля</th>
                <th className="px-2 py-2 text-right font-medium">Бесплатно</th>
              </tr>
            </thead>
            <tbody>
              {current.byService.map((r) => {
                const p = prevSvc.get(r.name);
                return (
                  <tr key={r.name} className="border-b border-slate-100 last:border-0">
                    <td className="max-w-[160px] truncate py-2 pr-2 text-slate-700" title={r.name}>
                      {r.name}
                    </td>
                    <td className="px-2 py-2 text-right tabular-nums">
                      <div>{num(r.orders)}</div>
                      <div className="mt-0.5 flex justify-end">
                        <DeltaBadge d={delta(r.orders, p?.orders ?? 0)} />
                      </div>
                    </td>
                    <td className="px-2 py-2 text-right tabular-nums text-slate-500">{pct(r.share)}</td>
                    <td className="px-2 py-2 text-right tabular-nums">
                      <div>{num(r.paidOrders)}</div>
                      <div className="mt-0.5 flex justify-end">
                        <DeltaBadge d={delta(r.paidOrders, p?.paidOrders ?? 0)} />
                      </div>
                    </td>
                    <td className="px-2 py-2 text-right tabular-nums text-slate-500">
                      {rate(r.paidOrders, svcTotal.paid)}
                    </td>
                    <td className="px-2 py-2 text-right tabular-nums">{pct(r.freeDeliveryRate)}</td>
                  </tr>
                );
              })}
            </tbody>
            <tfoot>
              <tr className="border-t-2 border-slate-200 font-semibold text-slate-700">
                <td className="py-2 pr-2">Итого</td>
                <td className="px-2 py-2 text-right tabular-nums">
                  <div>{num(svcTotal.orders)}</div>
                  <div className="mt-0.5 flex justify-end">
                    <DeltaBadge d={delta(svcTotal.orders, svcTotalPrev.orders)} />
                  </div>
                </td>
                <td className="px-2 py-2 text-right tabular-nums text-slate-500">{pct(100)}</td>
                <td className="px-2 py-2 text-right tabular-nums">
                  <div>{num(svcTotal.paid)}</div>
                  <div className="mt-0.5 flex justify-end">
                    <DeltaBadge d={delta(svcTotal.paid, svcTotalPrev.paid)} />
                  </div>
                </td>
                <td className="px-2 py-2 text-right tabular-nums text-slate-500">{pct(100)}</td>
                <td className="px-2 py-2 text-right tabular-nums">{rate(svcTotal.free, svcTotal.orders)}</td>
              </tr>
            </tfoot>
          </table>
      </div>

      <div className="overflow-x-auto rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-slate-500">По городам</h2>
        <table className="w-full min-w-[720px] text-sm">
          <thead>
            <tr className="border-b border-slate-200 text-left text-xs uppercase tracking-wide text-slate-400">
              <th className="py-2 pr-2 font-medium">Город</th>
              <th className="px-2 py-2 text-right font-medium">Заказы</th>
              <th className="px-2 py-2 text-right font-medium">Доля</th>
              <th className="px-2 py-2 text-right font-medium">Выручка</th>
              <th className="px-2 py-2 text-right font-medium">Оплата</th>
              <th className="px-2 py-2 text-right font-medium">Бесплатно</th>
            </tr>
          </thead>
          <tbody>
            {current.byCity.map((r) => {
              const p = prevCity.get(r.name);
              return (
                <tr
                  key={r.name}
                  className={`border-b border-slate-100 last:border-0 ${r.isPilot ? "bg-indigo-50/40" : ""}`}
                >
                  <td className="py-2 pr-2 text-slate-700">
                    {r.name}
                    {r.isPilot && (
                      <span className="ml-2 rounded-full bg-indigo-100 px-1.5 py-0.5 text-[10px] font-medium text-indigo-800">
                        пилот
                      </span>
                    )}
                  </td>
                  <td className="px-2 py-2 text-right tabular-nums">
                    <div>{num(r.orders)}</div>
                    <div className="mt-0.5 flex justify-end">
                      <DeltaBadge d={delta(r.orders, p?.orders ?? 0)} />
                    </div>
                  </td>
                  <td className="px-2 py-2 text-right tabular-nums text-slate-500">{pct(r.share)}</td>
                  <td className="px-2 py-2 text-right tabular-nums">
                    <div>{rub(r.revenue)}</div>
                    <div className="mt-0.5 flex justify-end">
                      <DeltaBadge d={delta(r.revenue, p?.revenue ?? 0)} />
                    </div>
                  </td>
                  <td className="px-2 py-2 text-right tabular-nums">
                    <div>{pct(r.paidRate)}</div>
                    <div className="mt-0.5 flex justify-end">
                      <DeltaBadge d={delta(r.paidRate, p?.paidRate ?? 0)} />
                    </div>
                  </td>
                  <td className="px-2 py-2 text-right tabular-nums">{pct(r.freeDeliveryRate)}</td>
                </tr>
              );
            })}
          </tbody>
          <tfoot>
            <tr className="border-t-2 border-slate-200 font-semibold text-slate-700">
              <td className="py-2 pr-2">Итого</td>
              <td className="px-2 py-2 text-right tabular-nums">
                <div>{num(cityTotal.orders)}</div>
                <div className="mt-0.5 flex justify-end">
                  <DeltaBadge d={delta(cityTotal.orders, cityTotalPrev.orders)} />
                </div>
              </td>
              <td className="px-2 py-2 text-right tabular-nums text-slate-500">{pct(100)}</td>
              <td className="px-2 py-2 text-right tabular-nums">
                <div>{rub(cityTotal.revenue)}</div>
                <div className="mt-0.5 flex justify-end">
                  <DeltaBadge d={delta(cityTotal.revenue, cityTotalPrev.revenue)} />
                </div>
              </td>
              <td className="px-2 py-2 text-right tabular-nums">
                <div>{rate(cityTotal.paid, cityTotal.orders)}</div>
                <div className="mt-0.5 flex justify-end">
                  <DeltaBadge
                    d={delta(
                      prevPaidRate(cityTotal.orders, cityTotal.paid),
                      prevPaidRate(cityTotalPrev.orders, cityTotalPrev.paid)
                    )}
                  />
                </div>
              </td>
              <td className="px-2 py-2 text-right tabular-nums">{rate(cityTotal.free, cityTotal.orders)}</td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>
  );
}
