import type { NamedCount } from "../types";
import { num, rub } from "../utils/format";

interface Props {
  title: string;
  rows: NamedCount[];
  metric?: "orders" | "revenue";
}

export default function BreakdownList({ title, rows, metric = "orders" }: Props) {
  const max = Math.max(1, ...rows.map((r) => (metric === "revenue" ? r.revenue : r.orders)));
  return (
    <div className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
      <h2 className="mb-4 text-sm font-semibold uppercase tracking-wide text-slate-500">{title}</h2>
      <div className="space-y-2.5">
        {rows.length === 0 && <div className="text-sm text-slate-400">Нет данных</div>}
        {rows.map((r) => {
          const v = metric === "revenue" ? r.revenue : r.orders;
          return (
            <div key={r.name}>
              <div className="flex items-baseline justify-between text-sm">
                <span className="truncate pr-2 text-slate-700">{r.name}</span>
                <span className="shrink-0 font-medium text-ink">
                  {num(r.orders)} зак. · {rub(r.revenue)}
                </span>
              </div>
              <div className="mt-1 h-1.5 overflow-hidden rounded bg-slate-100">
                <div className="h-full bg-brand" style={{ width: `${(v / max) * 100}%` }} />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
