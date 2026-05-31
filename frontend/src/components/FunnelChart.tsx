import { useState } from "react";
import type { FunnelStep } from "../types";
import { num, pct, rub } from "../utils/format";

interface Props {
  stages: FunnelStep[];
}

const stepColor = [
  "bg-slate-400",
  "bg-emerald-500",
  "bg-teal-500",
  "bg-indigo-500",
  "bg-violet-500",
  "bg-fuchsia-600",
];

type Metric = "orders" | "revenue" | "units";

const metrics: { key: Metric; label: string }[] = [
  { key: "orders", label: "Заказы" },
  { key: "revenue", label: "Выручка" },
  { key: "units", label: "Товары" },
];

function round1(n: number): number {
  return Math.round(n * 10) / 10;
}

export default function FunnelChart({ stages }: Props) {
  const [metric, setMetric] = useState<Metric>("orders");

  const valueOf = (s: FunnelStep) => s[metric];
  const fmt = (n: number) => (metric === "revenue" ? rub(n) : num(n));
  const base = stages.length > 0 ? valueOf(stages[0]) : 0;

  return (
    <div className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">
          Путь заказа (кумулятивно)
        </h2>
        <div className="flex gap-1 rounded-lg bg-slate-100 p-0.5">
          {metrics.map((m) => (
            <button
              key={m.key}
              onClick={() => setMetric(m.key)}
              className={`rounded-md px-2.5 py-1 text-xs font-medium transition ${
                metric === m.key
                  ? "bg-white text-ink shadow-sm"
                  : "text-slate-500 hover:text-ink"
              }`}
            >
              {m.label}
            </button>
          ))}
        </div>
      </div>
      <div className="space-y-1">
        {stages.map((s, i) => {
          const prev = i > 0 ? stages[i - 1] : null;
          const cur = valueOf(s);
          const prevVal = prev ? valueOf(prev) : 0;
          const fromStart = base > 0 ? round1((cur / base) * 100) : 0;
          const fromPrev = i === 0 ? 100 : prevVal > 0 ? round1((cur / prevVal) * 100) : 0;
          const drop = prev ? prevVal - cur : 0;
          const leak = prev !== null && fromPrev < 90;
          return (
            <div key={s.key}>
              {prev && (
                <div className="flex items-center gap-2 py-0.5 pl-2 text-xs">
                  <span className="text-slate-300">↓</span>
                  <span className={leak ? "font-medium text-rose-600" : "text-slate-400"}>
                    шаг {pct(fromPrev)}
                    {drop > 0 && ` · −${fmt(drop)}`}
                    {leak && " — точка отвала"}
                  </span>
                </div>
              )}
              <div className="flex items-center gap-3">
                <div className="w-40 shrink-0 text-sm text-slate-600">{s.label}</div>
                <div className="relative h-9 flex-1 overflow-hidden rounded bg-slate-100">
                  <div
                    className={`flex h-full items-center justify-end rounded pr-2 ${stepColor[i % stepColor.length]}`}
                    style={{ width: `${Math.max(fromStart, 6)}%` }}
                  >
                    <span className="text-xs font-semibold text-white">{fmt(cur)}</span>
                  </div>
                </div>
                <div className="w-16 shrink-0 text-right text-sm font-medium text-ink">{pct(fromStart)}</div>
              </div>
            </div>
          );
        })}
      </div>
      <p className="mt-3 text-xs text-slate-400">
        Стадии кумулятивны: заказ учитывается, если дошёл хотя бы до этой стадии. Стадии после
        «Отправлен» зависят и от того, что часть свежих заказов ещё в пути.
      </p>
    </div>
  );
}
