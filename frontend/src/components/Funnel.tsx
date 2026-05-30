import type { FunnelStage } from "../types";
import { num } from "../utils/format";

interface Props {
  stages: FunnelStage[];
}

const stageColor: Record<string, string> = {
  new: "bg-slate-400",
  processing: "bg-sky-400",
  shipped: "bg-indigo-400",
  in_pvz: "bg-violet-400",
  completed: "bg-emerald-500",
  closed: "bg-slate-500",
  returned: "bg-amber-500",
  canceled: "bg-rose-500",
};

export default function Funnel({ stages }: Props) {
  const max = Math.max(1, ...stages.map((s) => s.orders));
  return (
    <div className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
      <h2 className="mb-4 text-sm font-semibold uppercase tracking-wide text-slate-500">
        Воронка по статусам
      </h2>
      <div className="space-y-2">
        {stages.length === 0 && <div className="text-sm text-slate-400">Нет данных за период</div>}
        {stages.map((s) => (
          <div key={s.stage} className="flex items-center gap-3">
            <div className="w-28 shrink-0 text-sm text-slate-600">{s.label}</div>
            <div className="h-6 flex-1 overflow-hidden rounded bg-slate-100">
              <div
                className={`h-full ${stageColor[s.stage] ?? "bg-slate-400"}`}
                style={{ width: `${(s.orders / max) * 100}%` }}
              />
            </div>
            <div className="w-12 shrink-0 text-right text-sm font-medium text-ink">{num(s.orders)}</div>
          </div>
        ))}
      </div>
    </div>
  );
}
