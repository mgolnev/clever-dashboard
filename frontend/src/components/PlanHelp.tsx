import { useState } from "react";

// PlanHelp — сворачиваемая подсказка «Как пользоваться» для вкладки «Цель».
export default function PlanHelp() {
  const [open, setOpen] = useState(false);

  return (
    <div className="text-sm">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-brand transition hover:bg-brand/5"
      >
        <span className="flex h-4 w-4 items-center justify-center rounded-full bg-brand text-[11px] font-bold text-white">
          ?
        </span>
        Как пользоваться
      </button>

      {open && (
        <div className="mt-2 space-y-3 rounded-lg bg-slate-50 p-4 text-slate-600 ring-1 ring-slate-200">
          <p>
            Вкладка считает, <strong>сколько нужно трафика</strong>, чтобы выполнить план
            продаж (NET — выкупленная выручка) при текущей конверсии, и показывает,
            успеваете ли вы.
          </p>
          <ol className="list-decimal space-y-1.5 pl-5">
            <li>
              Выберите <strong>год</strong> и в таблице заполните по месяцам{" "}
              <strong>план NET</strong> (всего / сайт / приложение) и{" "}
              <strong>трафик</strong> (визиты сайт / приложение). Нажмите{" "}
              <strong>«Сохранить»</strong>.
            </li>
            <li>
              Выберите <strong>месяц для анализа</strong> — в блоке «Достижение плана»
              появятся факт, разрыв до плана и <strong>нужный трафик</strong> (всего, в
              день и сколько в день нужно, чтобы догнать).
            </li>
            <li>
              В блоке <strong>«What-if»</strong> покрутите рычаги (визиты, CR, чек,
              выкупаемость) — прогноз сразу покажет, добиваете ли план.
            </li>
          </ol>
          <p className="text-xs text-slate-500">
            Логика: <strong>NET = Визиты × CR × AOV × Выкупаемость</strong>. Отсюда нужный
            трафик = План ÷ (CR × AOV × R). «—» или «введите трафик» означает, что трафик
            не заполнен или ещё нет заказов за месяц. Трафик пока вводится вручную; загрузка
            из Яндекс.Метрики и AppMetrica — в планах.
          </p>
        </div>
      )}
    </div>
  );
}
