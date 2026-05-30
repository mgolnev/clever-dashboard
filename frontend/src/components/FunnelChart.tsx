import type { FunnelStep } from "../types";
import { num, pct } from "../utils/format";

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

export default function FunnelChart({ stages }: Props) {
  return (
    <div className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
      <h2 className="mb-4 text-sm font-semibold uppercase tracking-wide text-slate-500">
        Путь заказа (кумулятивно)
      </h2>
      <div className="space-y-1">
        {stages.map((s, i) => {
          const prev = i > 0 ? stages[i - 1] : null;
          const drop = prev ? prev.orders - s.orders : 0;
          const leak = prev && s.fromPrev < 90;
          return (
            <div key={s.key}>
              {prev && (
                <div className="flex items-center gap-2 py-0.5 pl-2 text-xs">
                  <span className="text-slate-300">↓</span>
                  <span className={leak ? "font-medium text-rose-600" : "text-slate-400"}>
                    шаг {pct(s.fromPrev)}
                    {drop > 0 && ` · −${num(drop)} зак.`}
                    {leak && " — точка отвала"}
                  </span>
                </div>
              )}
              <div className="flex items-center gap-3">
                <div className="w-40 shrink-0 text-sm text-slate-600">{s.label}</div>
                <div className="relative h-9 flex-1 overflow-hidden rounded bg-slate-100">
                  <div
                    className={`flex h-full items-center justify-end rounded pr-2 ${stepColor[i % stepColor.length]}`}
                    style={{ width: `${Math.max(s.fromStart, 6)}%` }}
                  >
                    <span className="text-xs font-semibold text-white">{num(s.orders)}</span>
                  </div>
                </div>
                <div className="w-16 shrink-0 text-right text-sm font-medium text-ink">{pct(s.fromStart)}</div>
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
