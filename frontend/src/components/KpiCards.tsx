import type { KPI } from "../types";
import { delta, num, pct, rub } from "../utils/format";
import DeltaBadge from "./DeltaBadge";

interface Props {
  current: KPI;
  prev: KPI;
}

interface CardDef {
  label: string;
  value: string;
  cur: number;
  prv: number;
  invert?: boolean;
  hint?: string;
}

export default function KpiCards({ current, prev }: Props) {
  const cards: CardDef[] = [
    { label: "Выручка", value: rub(current.revenue), cur: current.revenue, prv: prev.revenue },
    { label: "Заказы", value: num(current.orders), cur: current.orders, prv: prev.orders },
    { label: "Средний чек", value: rub(current.aov), cur: current.aov, prv: prev.aov, hint: "AOV на заказ" },
    { label: "Ср. цена позиции", value: rub(current.asp), cur: current.asp, prv: prev.asp, hint: "ASP за единицу" },
    {
      label: "Оплачено",
      value: pct(current.paidRate),
      cur: current.paidRate,
      prv: prev.paidRate,
      hint: `${num(current.paidOrders)} заказов`,
    },
    {
      label: "Отмены",
      value: pct(current.canceledRate),
      cur: current.canceledRate,
      prv: prev.canceledRate,
      invert: true,
      hint: `${num(current.canceledOrders)} заказов`,
    },
    {
      label: "Выкуп / оформл. (G2N)",
      value: pct(current.g2n),
      cur: current.g2n,
      prv: prev.g2n,
      hint: `${num(current.completed)} выкуплено`,
    },
    {
      label: "Выкупаемость",
      value: pct(current.redemptionRate),
      cur: current.redemptionRate,
      prv: prev.redemptionRate,
      hint: `из ${num(current.terminal)} в конечном статусе`,
    },
    {
      label: "Заказы в пути",
      value: num(current.inTransit),
      cur: current.inTransit,
      prv: prev.inTransit,
      hint: "ещё не выкуп/отмена",
    },
    { label: "Товаров продано", value: num(current.units), cur: current.units, prv: prev.units },
    { label: "Покупатели", value: num(current.customers), cur: current.customers, prv: prev.customers },
  ];

  return (
    <div className="grid grid-cols-2 gap-4 md:grid-cols-4 xl:grid-cols-4">
      {cards.map((c) => (
        <div key={c.label} className="rounded-xl bg-white p-4 shadow-sm ring-1 ring-slate-200">
          <div className="text-xs font-medium uppercase tracking-wide text-slate-500">{c.label}</div>
          <div className="mt-1 text-xl font-semibold text-ink">{c.value}</div>
          <div className="mt-2 flex items-center gap-2">
            <DeltaBadge d={delta(c.cur, c.prv)} invert={c.invert} />
            {c.hint && <span className="text-xs text-slate-400">{c.hint}</span>}
          </div>
        </div>
      ))}
    </div>
  );
}
