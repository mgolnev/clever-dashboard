import type { KPI, KPIStages, StageKPI } from "../types";
import { delta, num, pct, rub } from "../utils/format";
import DeltaBadge from "./DeltaBadge";

interface Props {
  current: KPI;
  prev: KPI;
}

type StageKey = keyof KPIStages;

const STAGES: { key: StageKey; label: string }[] = [
  { key: "created", label: "Оформлено" },
  { key: "paid", label: "Оплачено" },
  { key: "inTransit", label: "Транзит" },
  { key: "completed", label: "Выкуплено" },
];

interface MetricDef {
  label: string;
  hint?: string;
  // additive: метрика суммируема — для неё считаем долю от «Оформлено».
  additive?: boolean;
  pick: (s: StageKPI) => number;
  fmt: (s: StageKPI) => string;
}

const METRICS: MetricDef[] = [
  { label: "Выручка", additive: true, pick: (s) => s.revenue, fmt: (s) => rub(s.revenue) },
  { label: "Заказы", additive: true, pick: (s) => s.orders, fmt: (s) => num(s.orders) },
  { label: "Товары", hint: "проданные единицы", additive: true, pick: (s) => s.units, fmt: (s) => num(s.units) },
  { label: "Средний чек", hint: "AOV на заказ", pick: (s) => s.aov, fmt: (s) => rub(s.aov) },
  { label: "ASP", hint: "цена за единицу", pick: (s) => s.asp, fmt: (s) => rub(s.asp) },
  { label: "UPT", hint: "позиций на заказ", pick: (s) => s.upt, fmt: (s) => s.upt.toFixed(2) },
];

function sharePct(value: number, base: number): string {
  if (base <= 0) return "—";
  return `${Math.round((value / base) * 100)}%`;
}

// Коэффициенты выкупа: G2N (к оформленным), P2N (к оплаченным) и доля возврата
// оплаченных (оплачены, но не выкуплены среди дошедших до конечного статуса).
interface RatioDef {
  label: string;
  hint: string;
  num: StageKey;
  den: StageKey;
  complement?: boolean; // показывать 100% − доля (например, % возврата)
  invert?: boolean; // рост = плохо
}

const RATIOS: RatioDef[] = [
  { label: "G2N всего", hint: "выкуп / оформлено", num: "completed", den: "created" },
  { label: "G2N в кон.", hint: "выкуп / в конечном статусе", num: "completed", den: "terminal" },
  { label: "P2N всего", hint: "выкуп / оплачено", num: "completed", den: "paid" },
  {
    label: "Возврат опл.",
    hint: "оплачены, но не выкуплены (отмена/возврат) среди дошедших до конца",
    num: "completed",
    den: "paidTerminal",
    complement: true,
    invert: true,
  },
];

function ratioValue(stages: KPIStages, m: MetricDef, rt: RatioDef): number {
  const d = m.pick(stages[rt.den]);
  if (d <= 0) return 0;
  const r = (m.pick(stages[rt.num]) / d) * 100;
  return rt.complement ? 100 - r : r;
}

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
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {METRICS.map((m) => (
          <div key={m.label} className="rounded-xl bg-white p-4 shadow-sm ring-1 ring-slate-200">
            <div className="flex items-baseline justify-between">
              <span className="text-xs font-semibold uppercase tracking-wide text-ink">{m.label}</span>
              {m.hint && <span className="text-[11px] text-slate-400">{m.hint}</span>}
            </div>
            <div className="mt-3 space-y-2">
              {STAGES.map((st) => {
                const cs = current.stages[st.key];
                const ps = prev.stages[st.key];
                const base = m.pick(current.stages.created);
                return (
                  <div key={st.key} className="flex items-center justify-between gap-2">
                    <span className="w-[68px] shrink-0 text-xs font-medium text-ink">{st.label}</span>
                    <span className="w-10 shrink-0 text-right text-xs tabular-nums text-ink">
                      {m.additive ? sharePct(m.pick(cs), base) : ""}
                    </span>
                    <span className="flex-1 text-right text-sm font-semibold text-ink">{m.fmt(cs)}</span>
                    <span className="w-[72px] shrink-0 text-right">
                      <DeltaBadge d={delta(m.pick(cs), m.pick(ps))} />
                    </span>
                  </div>
                );
              })}
            </div>
            {m.additive && (
              <div className="mt-2 space-y-1.5 border-t border-slate-100 pt-2">
                {RATIOS.map((rt) => {
                  const cv = ratioValue(current.stages, m, rt);
                  const pv = ratioValue(prev.stages, m, rt);
                  return (
                    <div key={rt.label} className="flex items-center justify-between gap-2" title={rt.hint}>
                      <span className="w-[72px] shrink-0 text-xs font-medium text-ink">{rt.label}</span>
                      <span className="flex-1 text-right text-sm font-medium text-ink">{pct(cv)}</span>
                      <span className="w-[72px] shrink-0 text-right">
                        <DeltaBadge d={delta(cv, pv)} invert={rt.invert} />
                      </span>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        ))}
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
        {rates.map((c) => (
          <div key={c.label} className="rounded-xl bg-white p-3 shadow-sm ring-1 ring-slate-200">
            <div className="text-[11px] font-medium uppercase tracking-wide text-slate-500">{c.label}</div>
            <div className="mt-1 text-lg font-semibold text-ink">{c.value}</div>
            <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
              <DeltaBadge d={delta(c.cur, c.prv)} invert={c.invert} />
              {c.hint && <span className="text-[11px] text-slate-400">{c.hint}</span>}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
