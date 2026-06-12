import { useEffect, useState } from "react";
import type { ChannelGoal } from "../utils/planCalc";
import { forecastNet, type ChannelMetrics } from "../utils/planCalc";
import { pct, rub } from "../utils/format";

interface Props {
  goals: ChannelGoal[];
}

type LeverState = {
  visits: string;
  cr: string;
  aov: string;
  r: string;
};

function parseNum(s: string): number {
  const n = parseFloat(s.replace(",", "."));
  return Number.isFinite(n) ? n : 0;
}

function metricsFromLevers(levers: LeverState): ChannelMetrics {
  const visits = parseNum(levers.visits);
  const cr = parseNum(levers.cr) / 100;
  const aov = parseNum(levers.aov);
  const r = parseNum(levers.r) / 100;
  const orders = visits * cr;
  const revenue = orders * aov;
  const netRevenue = revenue * r;
  return { visits, orders, revenue, netRevenue, cr, aov, r };
}

function WhatIfChannel({ goal }: { goal: ChannelGoal }) {
  const [levers, setLevers] = useState<LeverState>({
    visits: String(goal.factVisits),
    cr: String((goal.metrics.cr * 100).toFixed(2)),
    aov: String(Math.round(goal.metrics.aov)),
    r: String((goal.metrics.r * 100).toFixed(1)),
  });

  useEffect(() => {
    setLevers({
      visits: String(goal.factVisits),
      cr: String((goal.metrics.cr * 100).toFixed(2)),
      aov: String(Math.round(goal.metrics.aov)),
      r: String((goal.metrics.r * 100).toFixed(1)),
    });
  }, [goal]);

  const m = metricsFromLevers(levers);
  const forecast = Math.round(forecastNet(m));
  const plan = goal.planTarget;
  const diff = forecast - plan;
  const onTrack = diff >= 0;

  const fields: { key: keyof LeverState; label: string; hint: string }[] = [
    { key: "visits", label: "Визиты", hint: "за месяц" },
    { key: "cr", label: "CR, %", hint: "заказы / визиты" },
    { key: "aov", label: "AOV, ₽", hint: "средний чек" },
    { key: "r", label: "R, %", hint: "выкупаемость" },
  ];

  return (
    <div className="rounded-lg bg-white p-4 ring-1 ring-slate-100">
      <h4 className="mb-3 text-sm font-semibold text-ink">{goal.label}</h4>
      <div className="mb-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
        {fields.map(({ key, label, hint }) => (
          <label key={key} className="block">
            <span className="text-xs text-slate-500">{label}</span>
            <input
              type="number"
              min={0}
              step={key === "cr" || key === "r" ? 0.01 : 1}
              value={levers[key]}
              onChange={(e) => setLevers((s) => ({ ...s, [key]: e.target.value }))}
              className="mt-0.5 w-full rounded-md border border-slate-200 px-2 py-1.5 text-sm focus:border-brand focus:outline-none focus:ring-1 focus:ring-brand"
            />
            <span className="text-[10px] text-slate-400">{hint}</span>
          </label>
        ))}
      </div>
      <div className="flex flex-wrap items-baseline gap-x-4 gap-y-1 text-sm">
        <span>
          Прогноз NET: <strong>{rub(forecast)}</strong>
        </span>
        <span className="text-slate-500">План: {rub(plan)}</span>
        {plan > 0 && (
          <span className={onTrack ? "text-emerald-600" : "text-rose-600"}>
            {onTrack ? "добиваем" : "не добиваем"} на {rub(Math.abs(diff))}
          </span>
        )}
        {m.visits > 0 && (
          <span className="text-xs text-slate-400">
            CR {pct(m.cr * 100)} · AOV {rub(Math.round(m.aov))} · R {pct(m.r * 100)}
          </span>
        )}
      </div>
    </div>
  );
}

export default function WhatIf({ goals }: Props) {
  return (
    <div className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
      <h2 className="mb-1 text-base font-semibold text-ink">What-if: рычаги</h2>
      <p className="mb-4 text-xs text-slate-500">
        NET = визиты × CR × AOV × R. Измените рычаги — увидите прогноз относительно плана.
      </p>
      <div className="space-y-4">
        {goals.map((g) => (
          <WhatIfChannel key={g.channel} goal={g} />
        ))}
      </div>
    </div>
  );
}
