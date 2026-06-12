import type { KPI, StageKPI } from "../types";
import { num, pct, rub, rubAbs, numAbs, ppAbs, floatAbs } from "../utils/format";
import StageFunnelGrid, { DEFAULT_RATIOS, type StageMetricDef } from "./StageFunnelGrid";

interface Props {
  current: KPI;
  prev: KPI;
  showCompare?: boolean;
}

function perCustomer(s: StageKPI, value: number): number {
  return s.customers > 0 ? value / s.customers : 0;
}

const METRICS: StageMetricDef[] = [
  {
    label: "Покупатели",
    additive: true,
    pick: (s) => s.customers,
    fmt: (s) => num(s.customers),
    fmtAbs: numAbs,
  },
  {
    label: "Выручка на клиента",
    hint: "ARPU",
    additive: true,
    pick: (s) => perCustomer(s, s.revenue),
    fmt: (s) => (s.customers > 0 ? rub(Math.round(s.revenue / s.customers)) : "—"),
    fmtAbs: rubAbs,
  },
  {
    label: "Заказов на клиента",
    pick: (s) => perCustomer(s, s.orders),
    fmt: (s) => (s.customers > 0 ? perCustomer(s, s.orders).toFixed(2) : "—"),
    fmtAbs: floatAbs,
  },
  {
    label: "Товаров на клиента",
    hint: "ед. на покупателя",
    pick: (s) => perCustomer(s, s.units),
    fmt: (s) => (s.customers > 0 ? perCustomer(s, s.units).toFixed(2) : "—"),
    fmtAbs: floatAbs,
  },
  {
    label: "Средний чек",
    hint: "AOV на заказ",
    pick: (s) => s.aov,
    fmt: (s) => rub(s.aov),
    fmtAbs: rubAbs,
  },
  {
    label: "Выручка",
    additive: true,
    pick: (s) => s.revenue,
    fmt: (s) => rub(s.revenue),
    fmtAbs: rubAbs,
  },
];

function customerRate(num: number, den: number): number {
  return den > 0 ? (num / den) * 100 : 0;
}

export default function CustomerKpiCards({ current, prev, showCompare = true }: Props) {
  const curCreated = current.stages.created.customers;
  const prvCreated = prev.stages.created.customers;

  const rates = [
    {
      label: "Оплатили",
      value: pct(customerRate(current.stages.paid.customers, curCreated)),
      cur: customerRate(current.stages.paid.customers, curCreated),
      prv: customerRate(prev.stages.paid.customers, prvCreated),
      hint: `${num(current.stages.paid.customers)} клиент.`,
      fmtAbs: ppAbs,
    },
    {
      label: "Отменили",
      value: pct(customerRate(current.canceledCustomers, curCreated)),
      cur: customerRate(current.canceledCustomers, curCreated),
      prv: customerRate(prev.canceledCustomers, prvCreated),
      invert: true,
      hint: `${num(current.canceledCustomers)} клиент.`,
      fmtAbs: ppAbs,
    },
    {
      label: "G2N",
      value: pct(customerRate(current.stages.completed.customers, curCreated)),
      cur: customerRate(current.stages.completed.customers, curCreated),
      prv: customerRate(prev.stages.completed.customers, prvCreated),
      hint: "выкуп / оформл.",
      fmtAbs: ppAbs,
    },
    {
      label: "Выкупаемость",
      value: pct(customerRate(current.stages.completed.customers, current.stages.terminal.customers)),
      cur: customerRate(current.stages.completed.customers, current.stages.terminal.customers),
      prv: customerRate(prev.stages.completed.customers, prev.stages.terminal.customers),
      hint: `из ${num(current.stages.terminal.customers)} в конечн.`,
      fmtAbs: ppAbs,
    },
    {
      label: "Повторные",
      value: pct(customerRate(current.repeatCustomers, current.customers)),
      cur: customerRate(current.repeatCustomers, current.customers),
      prv: customerRate(prev.repeatCustomers, prev.customers),
      hint: `${num(current.repeatCustomers)} из ${num(current.customers)}`,
      fmtAbs: ppAbs,
    },
    {
      label: "Заказов на клиента",
      value: current.customers > 0 ? (current.orders / current.customers).toFixed(2) : "—",
      cur: current.customers > 0 ? current.orders / current.customers : 0,
      prv: prev.customers > 0 ? prev.orders / prev.customers : 0,
      fmtAbs: floatAbs,
    },
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
