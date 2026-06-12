import type { KPIStages, StageKPI } from "../types";
import { delta, pct, ppAbs } from "../utils/format";
import DeltaBadge from "./DeltaBadge";

type StageKey = keyof KPIStages;

const STAGES: { key: StageKey; label: string }[] = [
  { key: "created", label: "Оформлено" },
  { key: "paid", label: "Оплачено" },
  { key: "inTransit", label: "Транзит" },
  { key: "completed", label: "Выкуплено" },
];

export interface StageMetricDef {
  label: string;
  hint?: string;
  additive?: boolean;
  pick: (s: StageKPI) => number;
  fmt: (s: StageKPI) => string;
  fmtAbs?: (n: number) => string;
}

export interface RatioDef {
  label: string;
  hint: string;
  num: StageKey;
  den: StageKey;
  complement?: boolean;
  invert?: boolean;
}

export interface RateCardDef {
  label: string;
  value: string;
  cur: number;
  prv: number;
  invert?: boolean;
  hint?: string;
  fmtAbs?: (n: number) => string;
}

interface Props {
  currentStages: KPIStages;
  prevStages: KPIStages;
  metrics: StageMetricDef[];
  ratios?: RatioDef[];
  rates?: RateCardDef[];
  showCompare?: boolean;
}

function sharePct(value: number, base: number): string {
  if (base <= 0) return "—";
  return `${Math.round((value / base) * 100)}%`;
}

function ratioValue(stages: KPIStages, m: StageMetricDef, rt: RatioDef): number {
  const d = m.pick(stages[rt.den]);
  if (d <= 0) return 0;
  const r = (m.pick(stages[rt.num]) / d) * 100;
  return rt.complement ? 100 - r : r;
}

export default function StageFunnelGrid({
  currentStages,
  prevStages,
  metrics,
  ratios = [],
  rates = [],
  showCompare = true,
}: Props) {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4">
        {metrics.map((m) => (
          <div key={m.label} className="rounded-xl bg-white p-4 shadow-sm ring-1 ring-slate-200">
            <div className="flex items-baseline justify-between mb-2">
              <span className="text-xs font-semibold uppercase tracking-wide text-ink">{m.label}</span>
              {m.hint && <span className="text-[11px] text-slate-400">{m.hint}</span>}
            </div>
            <div className="mt-3 space-y-2">
              {STAGES.map((st) => {
                const cs = currentStages[st.key];
                const ps = prevStages[st.key];
                const base = m.pick(currentStages.created);
                return (
                  <div key={st.key} className="flex items-center justify-between gap-2 border-b border-slate-50 pb-1.5 last:border-0 last:pb-0">
                    <span className="w-[80px] shrink-0 text-xs font-medium text-ink">{st.label}</span>
                    <span className="w-12 shrink-0 text-right text-xs tabular-nums text-ink">
                      {m.additive ? sharePct(m.pick(cs), base) : ""}
                    </span>
                    <span className="flex-1 text-right text-sm font-semibold text-ink">{m.fmt(cs)}</span>
                    {showCompare && (
                      <>
                        <span className="w-[84px] shrink-0 text-right">
                          <DeltaBadge d={delta(m.pick(cs), m.pick(ps))} mode="pct" />
                        </span>
                        <span className="w-[120px] shrink-0 text-right">
                          <DeltaBadge d={delta(m.pick(cs), m.pick(ps))} mode="abs" fmtAbs={m.fmtAbs} />
                        </span>
                      </>
                    )}
                  </div>
                );
              })}
            </div>
            {m.additive && ratios.length > 0 && (
              <div className="mt-2 space-y-1.5 border-t border-slate-100 pt-2">
                {ratios.map((rt) => {
                  const cv = ratioValue(currentStages, m, rt);
                  const pv = ratioValue(prevStages, m, rt);
                  return (
                    <div key={rt.label} className="flex items-center justify-between gap-2 border-b border-slate-50/50 pb-1.5 last:border-0 last:pb-0" title={rt.hint}>
                      <span className="w-[80px] shrink-0 text-xs font-medium text-ink">{rt.label}</span>
                      <span className="flex-1 text-right text-sm font-medium text-ink">{pct(cv)}</span>
                      {showCompare && (
                        <>
                          <span className="w-[84px] shrink-0 text-right">
                            <DeltaBadge d={delta(cv, pv)} invert={rt.invert} mode="pct" />
                          </span>
                          <span className="w-[120px] shrink-0 text-right">
                            <DeltaBadge d={delta(cv, pv)} invert={rt.invert} mode="abs" fmtAbs={ppAbs} />
                          </span>
                        </>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        ))}
      </div>

      {rates.length > 0 && (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
          {rates.map((c) => (
            <div key={c.label} className="rounded-xl bg-white p-3 shadow-sm ring-1 ring-slate-200">
              <div className="text-[11px] font-medium uppercase tracking-wide text-slate-500">{c.label}</div>
              <div className="mt-1 text-lg font-semibold text-ink">{c.value}</div>
              <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
                {showCompare && (
                  <DeltaBadge d={delta(c.cur, c.prv)} invert={c.invert} fmtAbs={c.fmtAbs} />
                )}
                {c.hint && <span className="text-[11px] text-slate-400">{c.hint}</span>}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export const DEFAULT_RATIOS: RatioDef[] = [
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
