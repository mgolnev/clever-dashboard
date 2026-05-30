import type { ProductRow } from "../types";
import { num, rub } from "../utils/format";

interface Props {
  title: string;
  rows: ProductRow[];
  showUnits?: boolean;
}

export default function ProductTable({ title, rows, showUnits = true }: Props) {
  const max = Math.max(1, ...rows.map((r) => r.revenue));
  return (
    <div className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
      <h2 className="mb-4 text-sm font-semibold uppercase tracking-wide text-slate-500">{title}</h2>
      <div className="space-y-2.5">
        {rows.length === 0 && <div className="text-sm text-slate-400">Нет данных</div>}
        {rows.map((r) => (
          <div key={r.name}>
            <div className="flex items-baseline justify-between gap-2 text-sm">
              <span className="truncate text-slate-700" title={r.name}>
                {r.name}
              </span>
              <span className="shrink-0 font-medium text-ink">{rub(r.revenue)}</span>
            </div>
            <div className="mt-1 flex items-center gap-2">
              <div className="h-1.5 flex-1 overflow-hidden rounded bg-slate-100">
                <div className="h-full bg-emerald-500" style={{ width: `${(r.revenue / max) * 100}%` }} />
              </div>
              {showUnits && <span className="w-20 shrink-0 text-right text-xs text-slate-400">{num(r.units)} шт</span>}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
