import { useState } from "react";

export default function PlanHelp() {
  const [open, setOpen] = useState(false);

  return (
    <div className="relative text-sm">
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-expanded={open}
        className="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-brand transition hover:bg-indigo-50"
      >
        <span className="flex h-4 w-4 items-center justify-center rounded-full bg-brand text-[11px] font-bold text-white">?</span>
        Как считается
      </button>

      {open && (
        <div className="absolute right-0 z-20 mt-2 w-[min(32rem,calc(100vw-2rem))] rounded-xl bg-white p-4 text-slate-600 ring-1 ring-slate-200">
          <p className="font-medium text-ink">Вы задаёте только общую цель NET.</p>
          <ul className="mt-3 list-disc space-y-2 pl-5 text-xs leading-relaxed">
            <li>Сайт и приложение получают долю по фактической выручке последних доступных 30 дней перед месяцем.</li>
            <li>Факт NET — чистый выкуп: выкупленная выручка за вычетом возвратов.</li>
            <li>Трафик и CR приходят из Яндекс Метрики и AppMetrica, AOV — из заказов Битрикса.</li>
            <li>Прогноз учитывает ожидаемый G2N по зрелой истории и текущий дневной темп.</li>
          </ul>
        </div>
      )}
    </div>
  );
}
