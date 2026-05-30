import { useState } from "react";
import type { SegmentGroup } from "../types";
import { num, pct, rub } from "../utils/format";

interface Props {
  segments: SegmentGroup[];
}

function rateColor(rate: number, kind: "good" | "bad"): string {
  // good: выше = лучше (оплата, выкуп); bad: выше = хуже (отмена)
  const high = kind === "good" ? "text-emerald-600" : "text-rose-600";
  const low = kind === "good" ? "text-rose-600" : "text-emerald-600";
  if (kind === "good") return rate >= 80 ? high : rate >= 65 ? "text-amber-600" : low;
  return rate >= 35 ? high : rate >= 25 ? "text-amber-600" : low;
}

export default function SegmentFunnel({ segments }: Props) {
  const [active, setActive] = useState(segments[0]?.by ?? "payment");
  const group = segments.find((s) => s.by === active) ?? segments[0];

  return (
    <div className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">
          Воронка в разрезе
        </h2>
        <div className="flex flex-wrap gap-1.5">
          {segments.map((s) => (
            <button
              key={s.by}
              onClick={() => setActive(s.by)}
              className={`rounded-lg px-3 py-1 text-sm transition ${
                s.by === active
                  ? "bg-brand text-white"
                  : "border border-slate-300 text-slate-600 hover:border-brand hover:text-brand"
              }`}
            >
              {s.label}
            </button>
          ))}
        </div>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-200 text-left text-xs uppercase tracking-wide text-slate-400">
              <th className="py-2 pr-2 font-medium">{group?.label}</th>
              <th className="px-2 py-2 text-right font-medium">Гросс</th>
              <th className="px-2 py-2 text-right font-medium">Оплата</th>
              <th className="px-2 py-2 text-right font-medium">Отмена</th>
              <th className="px-2 py-2 text-right font-medium">Выкуп</th>
              <th className="px-2 py-2 text-right font-medium">Выручка</th>
            </tr>
          </thead>
          <tbody>
            {group?.rows.map((r) => (
              <tr key={r.name} className="border-b border-slate-100 last:border-0">
                <td className="max-w-[220px] truncate py-2 pr-2 text-slate-700" title={r.name}>
                  {r.name}
                  {r.problems > 0 && (
                    <span className="ml-2 rounded-full bg-amber-50 px-1.5 py-0.5 text-xs text-amber-700">
                      проблем: {r.problems}
                    </span>
                  )}
                </td>
                <td className="px-2 py-2 text-right tabular-nums text-slate-600">{num(r.gross)}</td>
                <td className={`px-2 py-2 text-right font-medium tabular-nums ${rateColor(r.paidRate, "good")}`}>
                  {pct(r.paidRate)}
                </td>
                <td className={`px-2 py-2 text-right font-medium tabular-nums ${rateColor(r.cancelRate, "bad")}`}>
                  {pct(r.cancelRate)}
                </td>
                <td className="px-2 py-2 text-right tabular-nums text-slate-600">{pct(r.completedRate)}</td>
                <td className="px-2 py-2 text-right tabular-nums text-slate-500">{rub(r.revenue)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="mt-3 text-xs text-slate-400">
        Оплата = оплачено / гросс. Отмена = отменено / гросс. Выкуп = «Выполнен» / гросс. Цвет: зелёный — здорово, красный — проблема.
      </p>
    </div>
  );
}
