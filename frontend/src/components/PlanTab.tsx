import { useState } from "react";
import { useGoalPlan } from "../hooks/useGoalPlan";
import GoalSummary from "./GoalSummary";
import PlanEditor from "./PlanEditor";
import PlanHelp from "./PlanHelp";
import WhatIf from "./WhatIf";

const MONTH_NAMES = [
  "Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
  "Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
];

export default function PlanTab() {
  const today = new Date();
  const currentYear = today.getFullYear();
  const [year, setYear] = useState(currentYear);
  const [month, setMonth] = useState(today.getMonth() + 1);
  const {
    plan,
    allocation,
    progress,
    analyticsStatus,
    loading,
    saving,
    error,
    saveTarget,
  } = useGoalPlan(year, month);
  const years = Array.from({ length: 5 }, (_, index) => currentYear - 2 + index);

  return (
    <div className="space-y-4">
      <section className="flex flex-wrap items-center gap-4 rounded-xl bg-white px-4 py-3 ring-1 ring-slate-200">
        <label className="flex items-center gap-2 text-sm">
          <span className="text-slate-500">Год</span>
          <select
            value={year}
            onChange={(event) => setYear(Number(event.target.value))}
            className="rounded-md border border-slate-200 px-2 py-1.5 outline-none focus:border-brand focus:ring-1 focus:ring-brand"
          >
            {years.map((value) => <option key={value} value={value}>{value}</option>)}
          </select>
        </label>
        <label className="flex items-center gap-2 text-sm">
          <span className="text-slate-500">Месяц</span>
          <select
            value={month}
            onChange={(event) => setMonth(Number(event.target.value))}
            className="rounded-md border border-slate-200 px-2 py-1.5 outline-none focus:border-brand focus:ring-1 focus:ring-brand"
          >
            {MONTH_NAMES.map((name, index) => <option key={name} value={index + 1}>{name}</option>)}
          </select>
        </label>
        <div className="ml-auto"><PlanHelp /></div>
      </section>

      {error && <div className="rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-700 ring-1 ring-rose-200">{error}</div>}

      {loading && !plan && (
        <div className="rounded-2xl bg-white px-6 py-12 text-center text-sm text-slate-400 ring-1 ring-slate-200">
          Загружаю цель и фактические показатели…
        </div>
      )}

      {plan && allocation && (
        <PlanEditor
          year={year}
          month={month}
          plan={plan}
          allocation={allocation}
          saving={saving}
          onSave={saveTarget}
          onSelectMonth={setMonth}
        />
      )}

      {loading && plan && (
        <div className="h-1 overflow-hidden rounded-full bg-slate-200">
          <div className="h-full w-1/3 animate-pulse rounded-full bg-brand" />
        </div>
      )}

      {progress && allocation && (
        <>
          <GoalSummary progress={progress} allocation={allocation} analyticsStatus={analyticsStatus} />
          {progress.target > 0 && <WhatIf progress={progress} />}
        </>
      )}
    </div>
  );
}
