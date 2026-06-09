import type { KPI } from "../types";
import { num, pct, rub } from "../utils/format";
import StageFunnelGrid, { DEFAULT_RATIOS, type StageMetricDef } from "./StageFunnelGrid";

interface Props {
  current: KPI;
  prev: KPI;
}

const METRICS: StageMetricDef[] = [
  { label: "Выручка", additive: true, pick: (s) => s.revenue, fmt: (s) => rub(s.revenue) },
  { label: "Заказы", additive: true, pick: (s) => s.orders, fmt: (s) => num(s.orders) },
  { label: "Товары", hint: "проданные единицы", additive: true, pick: (s) => s.units, fmt: (s) => num(s.units) },
  { label: "Средний чек", hint: "AOV на заказ", pick: (s) => s.aov, fmt: (s) => rub(s.aov) },
  { label: "ASP", hint: "цена за единицу", pick: (s) => s.asp, fmt: (s) => rub(s.asp) },
  { label: "UPT", hint: "позиций на заказ", pick: (s) => s.upt, fmt: (s) => s.upt.toFixed(2) },
];

export default function KpiCards({ current, prev }: Props) {
  const rates = [
    { label: "Оплачено", value: pct(current.paidRate), cur: current.paidRate, prv: prev.paidRate, hint: `${num(current.paidOrders)} зак.` },
    { label: "Отмены", value: pct(current.canceledRate), cur: current.canceledRate, prv: prev.canceledRate, invert: true, hint: `${num(current.canceledOrders)} зак.` },
    { label: "G2N", value: pct(current.g2n), cur: current.g2n, prv: prev.g2n, hint: "выкуп / оформл." },
    { label: "Выкупаемость", value: pct(current.redemptionRate), cur: current.redemptionRate, prv: prev.redemptionRate, hint: `из ${num(current.terminal)} в конечн.` },
    { label: "Заказы в пути", value: num(current.inTransit), cur: current.inTransit, prv: prev.inTransit, hint: "ещё не выкуп/отмена" },
    { label: "Покупатели", value: num(current.customers), cur: current.customers, prv: prev.customers },
  ];

  return (
    <StageFunnelGrid
      currentStages={current.stages}
      prevStages={prev.stages}
      metrics={METRICS}
      ratios={DEFAULT_RATIOS}
      rates={rates}
    />
  );
}
