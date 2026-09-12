import { useEffect, useMemo, useState } from "react";
import type { PlanReport } from "../types";
import { allocationTargets, type AllocationBasis } from "../utils/planCalc";
import { pct, rub } from "../utils/format";

const MONTH_NAMES = [
  "Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
  "Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
];

interface Props {
  year: number;
  month: number;
  plan: PlanReport;
  allocation: AllocationBasis;
  saving: boolean;
  onSave: (target: number) => Promise<void>;
  onSelectMonth: (month: number) => void;
}

function dateLabel(value: string): string {
  if (!value) return "";
  return new Date(`${value}T00:00:00`).toLocaleDateString("ru-RU", {
    day: "numeric",
    month: "short",
  });
}

export default function PlanEditor({ year, month, plan, allocation, saving, onSave, onSelectMonth }: Props) {
  const savedTarget = plan.months[month - 1].targets.all;
  const [draft, setDraft] = useState(String(savedTarget || ""));
  const parsedTarget = Math.max(0, Number(draft) || 0);
  const targets = useMemo(() => allocationTargets(parsedTarget, allocation), [parsedTarget, allocation]);

  useEffect(() => {
    setDraft(String(savedTarget || ""));
  }, [savedTarget, month, year]);

  const allocationText = allocation.source === "history"
    ? `по продажам за ${dateLabel(allocation.start)} — ${dateLabel(allocation.end)}`
    : allocation.source === "current"
      ? "по продажам выбранного месяца"
      : "поровну — исторических данных пока нет";

  return (
    <section className="overflow-hidden rounded-2xl bg-white ring-1 ring-slate-200">
      <div className="grid gap-0 lg:grid-cols-[1.25fr_0.75fr]">
        <div className="p-5 sm:p-7">
          <p className="text-xs font-semibold uppercase tracking-[0.14em] text-brand">Общая цель</p>
          <div className="mt-2 flex flex-wrap items-end gap-3">
            <label className="min-w-0 flex-1">
              <span className="sr-only">Цель NET на {MONTH_NAMES[month - 1]}</span>
              <div className="flex items-baseline border-b-2 border-slate-200 pb-2 focus-within:border-brand">
                <input
                  type="number"
                  min={0}
                  step={100000}
                  value={draft}
                  onChange={(event) => setDraft(event.target.value)}
                  placeholder="5 000 000"
                  className="min-w-0 flex-1 bg-transparent text-3xl font-semibold tabular-nums text-ink outline-none sm:text-4xl"
                />
                <span className="ml-2 text-xl font-medium text-slate-400">₽</span>
              </div>
            </label>
            <button
              type="button"
              onClick={() => void onSave(parsedTarget)}
              disabled={saving || parsedTarget === savedTarget}
              className="rounded-lg bg-brand px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-40"
            >
              {saving ? "Сохраняю…" : savedTarget > 0 ? "Обновить цель" : "Сохранить цель"}
            </button>
          </div>
          <p className="mt-2 text-sm text-slate-500">
            NET на {MONTH_NAMES[month - 1].toLowerCase()} {year}. Трафик, конверсии и разбивка по каналам рассчитываются автоматически.
          </p>
        </div>

        <div className="border-t border-slate-100 bg-slate-50 p-5 sm:p-7 lg:border-l lg:border-t-0">
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-xs font-medium uppercase tracking-wide text-slate-400">Автораспределение</p>
              <p className="mt-1 text-xs text-slate-500">{allocationText}</p>
            </div>
            <span className="rounded-full bg-white px-2.5 py-1 text-xs font-medium text-slate-500 ring-1 ring-slate-200">фиксируется при сохранении</span>
          </div>
          <div className="mt-5 space-y-4">
            <div>
              <div className="flex items-baseline justify-between gap-3 text-sm">
                <span className="font-medium text-ink">Сайт</span>
                <span className="tabular-nums text-slate-600">{rub(targets.site)} · {pct(allocation.siteShare * 100)}</span>
              </div>
              <div className="mt-1.5 h-1.5 overflow-hidden rounded-full bg-slate-200">
                <div className="h-full rounded-full bg-sky-500 transition-[width] duration-500" style={{ width: `${allocation.siteShare * 100}%` }} />
              </div>
            </div>
            <div>
              <div className="flex items-baseline justify-between gap-3 text-sm">
                <span className="font-medium text-ink">Приложение</span>
                <span className="tabular-nums text-slate-600">{rub(targets.app)} · {pct(allocation.appShare * 100)}</span>
              </div>
              <div className="mt-1.5 h-1.5 overflow-hidden rounded-full bg-slate-200">
                <div className="h-full rounded-full bg-violet-500 transition-[width] duration-500" style={{ width: `${allocation.appShare * 100}%` }} />
              </div>
            </div>
          </div>
        </div>
      </div>

      <details className="border-t border-slate-100">
        <summary className="cursor-pointer px-5 py-3 text-sm font-medium text-slate-600 hover:bg-slate-50 sm:px-7">
          Все цели на {year} год
        </summary>
        <div className="grid border-t border-slate-100 sm:grid-cols-2 lg:grid-cols-3">
          {plan.months.map((item) => (
            <button
              key={item.month}
              type="button"
              onClick={() => onSelectMonth(item.month)}
              className={`flex items-center justify-between gap-3 border-b border-slate-100 px-5 py-3 text-left transition hover:bg-slate-50 sm:px-7 ${item.month === month ? "bg-indigo-50/60" : ""}`}
            >
              <span className={item.month === month ? "font-semibold text-brand" : "text-slate-600"}>{MONTH_NAMES[item.month - 1]}</span>
              <span className="font-medium tabular-nums text-ink">{item.targets.all > 0 ? rub(item.targets.all) : "не задана"}</span>
            </button>
          ))}
        </div>
      </details>
    </section>
  );
}
