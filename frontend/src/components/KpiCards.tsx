import type { KPI } from "../types";
import { num, pct, rub, rubAbs, numAbs, ppAbs, floatAbs } from "../utils/format";
import StageFunnelGrid, { DEFAULT_RATIOS, type StageMetricDef } from "./StageFunnelGrid";

interface Props {
  current: KPI;
  prev: KPI;
  showCompare?: boolean;
}

const METRICS: StageMetricDef[] = [
  { label: "Выручка", additive: true, pick: (s) => s.revenue, fmt: (s) => rub(s.revenue), fmtAbs: rubAbs },
  { label: "Заказы", additive: true, pick: (s) => s.orders, fmt: (s) => num(s.orders), fmtAbs: numAbs },
  { label: "Товары", hint: "проданные единицы", additive: true, pick: (s) => s.units, fmt: (s) => num(s.units), fmtAbs: numAbs },
  { label: "Средний чек", hint: "AOV на заказ", pick: (s) => s.aov, fmt: (s) => rub(s.aov), fmtAbs: rubAbs },
  { label: "ASP", hint: "цена за единицу", pick: (s) => s.asp, fmt: (s) => rub(s.asp), fmtAbs: rubAbs },
  { label: "UPT", hint: "позиций на заказ", pick: (s) => s.upt, fmt: (s) => s.upt.toFixed(2), fmtAbs: floatAbs },
];

export default function KpiCards({ current, prev, showCompare = true }: Props) {
  const rates = [
    { label: "Оплачено", value: pct(current.paidRate), cur: current.paidRate, prv: prev.paidRate, hint: `${num(current.paidOrders)} зак.`, fmtAbs: ppAbs },
    { label: "Отмены", value: pct(current.canceledRate), cur: current.canceledRate, prv: prev.canceledRate, invert: true, hint: `${num(current.canceledOrders)} зак.`, fmtAbs: ppAbs },
    { label: "G2N", value: pct(current.g2n), cur: current.g2n, prv: prev.g2n, hint: "выкуп / оформл.", fmtAbs: ppAbs },
    { label: "Выкупаемость", value: pct(current.redemptionRate), cur: current.redemptionRate, prv: prev.redemptionRate, hint: `из ${num(current.terminal)} в конечн.`, fmtAbs: ppAbs },
    { label: "Заказы в пути", value: num(current.inTransit), cur: current.inTransit, prv: prev.inTransit, hint: "ещё не выкуп/отмена", fmtAbs: numAbs },
    { label: "Покупатели", value: num(current.customers), cur: current.customers, prv: prev.customers, fmtAbs: numAbs },
  ];

  return (
    <StageFunnelGrid
      currentStages={current.stages}
      prevStages={prev.stages}
      metrics={METRICS}
      ratios={DEFAULT_RATIOS}
      rates={rates}
      showCompare={showCompare}
    />
  );
}
