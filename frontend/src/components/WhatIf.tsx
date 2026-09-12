import { useEffect, useState } from "react";
import type { GoalProgress } from "../utils/planCalc";
import { forecastNet } from "../utils/planCalc";
import { pct, rub } from "../utils/format";

interface Props {
  progress: GoalProgress;
}

type LeverState = {
  sessions: string;
  cr: string;
  aov: string;
  g2n: string;
};

function parseNumber(value: string): number {
  const parsed = Number(value.replace(",", "."));
  return Number.isFinite(parsed) ? Math.max(0, parsed) : 0;
}

export default function WhatIf({ progress }: Props) {
  const [open, setOpen] = useState(false);
  const [levers, setLevers] = useState<LeverState>({ sessions: "", cr: "", aov: "", g2n: "" });

  useEffect(() => {
    setLevers({
      sessions: String(progress.driver.sessions),
      cr: String((progress.driver.netCr * 100).toFixed(2)),
      aov: String(Math.round(progress.driver.aov)),
      g2n: String((progress.driver.g2n * 100).toFixed(1)),
    });
  }, [progress.driver]);

  const sessions = parseNumber(levers.sessions);
  const cr = parseNumber(levers.cr) / 100;
  const aov = parseNumber(levers.aov);
  const g2n = parseNumber(levers.g2n) / 100;
  const forecast = Math.round(forecastNet(sessions, cr, aov, g2n));
  const difference = forecast - progress.target;

  return (
    <section className="rounded-2xl bg-white ring-1 ring-slate-200">
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-expanded={open}
        className="flex w-full items-center justify-between gap-4 px-5 py-4 text-left sm:px-7"
      >
        <div>
          <h2 className="font-semibold text-ink">Сценарий «что если»</h2>
          <p className="mt-0.5 text-xs text-slate-500">Проверьте, какой NET дадут другие значения трафика и конверсии.</p>
        </div>
        <span className={`text-lg text-slate-400 transition-transform ${open ? "rotate-180" : ""}`}>⌄</span>
      </button>

      {open && (
        <div className="border-t border-slate-100 px-5 py-5 sm:px-7">
          <div className="grid gap-4 sm:grid-cols-4">
            {([
              ["sessions", "Визиты / сессии", "за месяц"],
              ["cr", "CR в заказ, %", "неотменённые заказы / трафик"],
              ["aov", "AOV, ₽", "средний чек"],
              ["g2n", "G2N, %", "доля чистого выкупа"],
            ] as Array<[keyof LeverState, string, string]>).map(([key, label, hint]) => (
              <label key={key}>
                <span className="text-xs font-medium text-slate-600">{label}</span>
                <input
                  type="number"
                  min={0}
                  step={key === "cr" || key === "g2n" ? 0.1 : 1}
                  value={levers[key]}
                  onChange={(event) => setLevers((current) => ({ ...current, [key]: event.target.value }))}
                  className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm tabular-nums outline-none transition focus:border-brand focus:ring-2 focus:ring-indigo-100"
                />
                <span className="mt-1 block text-[11px] text-slate-400">{hint}</span>
              </label>
            ))}
          </div>
          <div className="mt-5 flex flex-wrap items-baseline gap-x-5 gap-y-2 border-t border-slate-100 pt-4">
            <span className="text-sm text-slate-500">Прогноз NET</span>
            <strong className="text-2xl tabular-nums text-ink">{rub(forecast)}</strong>
            {progress.target > 0 && (
              <span className={difference >= 0 ? "text-sm text-emerald-600" : "text-sm text-rose-600"}>
                {difference >= 0 ? `выше цели на ${rub(difference)}` : `ниже цели на ${rub(Math.abs(difference))}`}
              </span>
            )}
            <span className="text-xs text-slate-400">CR {pct(cr * 100)} · G2N {pct(g2n * 100)}</span>
          </div>
        </div>
      )}
    </section>
  );
}
