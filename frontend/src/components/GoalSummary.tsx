import type { AnalyticsStatus } from "../types";
import type { AllocationBasis, GoalProgress } from "../utils/planCalc";
import { num, pct, pct2, rub } from "../utils/format";

interface Props {
  progress: GoalProgress;
  allocation: AllocationBasis;
  analyticsStatus: AnalyticsStatus | null;
}

function shortDate(value?: string | null): string {
  if (!value) return "—";
  return new Date(`${value}T00:00:00`).toLocaleDateString("ru-RU", { day: "numeric", month: "long" });
}

function valueOrDash(value: number | null, format: (value: number) => string): string {
  return value == null ? "—" : format(value);
}

function SourceNote({ status, dataThrough }: { status: AnalyticsStatus | null; dataThrough: string | null }) {
  const metrika = status?.sources.find((source) => source.source === "metrika");
  const appmetrica = status?.sources.find((source) => source.source === "appmetrica");
  const trafficReady = Boolean(metrika?.lastDataDay || appmetrica?.lastDataDay);

  return (
    <div className="flex flex-wrap items-center gap-x-5 gap-y-2 border-b border-slate-200 pb-3 text-xs text-slate-500">
      <span><strong className="font-medium text-slate-700">Заказы:</strong> {dataThrough ? `по ${shortDate(dataThrough)}` : "нет данных за месяц"}</span>
      <span>
        <strong className="font-medium text-slate-700">Трафик:</strong>{" "}
        {trafficReady
          ? `Метрика по ${shortDate(metrika?.lastDataDay)}, AppMetrica по ${shortDate(appmetrica?.lastDataDay)}`
          : "автоматические данные не загружены"}
      </span>
    </div>
  );
}

export default function GoalSummary({ progress, allocation, analyticsStatus }: Props) {
  if (progress.target <= 0) {
    return (
      <section className="rounded-2xl border border-dashed border-slate-300 bg-white px-6 py-10 text-center">
        <h2 className="text-lg font-semibold text-ink">Цель на месяц ещё не задана</h2>
        <p className="mx-auto mt-2 max-w-xl text-sm text-slate-500">
          Укажите одну общую сумму выше. После сохранения здесь появятся темп, прогноз и необходимые показатели трафика.
        </p>
      </section>
    );
  }

  const progressWidth = Math.min(100, Math.max(0, progress.completionPct));
  const ahead = progress.paceDelta >= 0;
  const forecastOnTrack = progress.forecastNet != null && progress.forecastNet >= progress.target;
  const incompleteCoverage = Boolean(progress.coverageStarts && !progress.coverageStarts.endsWith("-01"));

  return (
    <div className="space-y-4">
      <section className="overflow-hidden rounded-2xl bg-slate-900 text-white">
        <div className="p-5 sm:p-7">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.14em] text-indigo-300">Выполнение плана</p>
              <div className="mt-3 flex flex-wrap items-baseline gap-x-3 gap-y-1">
                <strong className="text-3xl font-semibold tabular-nums sm:text-5xl">{rub(progress.factNet)}</strong>
                <span className="text-lg text-slate-400">из {rub(progress.target)}</span>
              </div>
            </div>
            {progress.dataThrough ? (
              <div className={`rounded-full px-3 py-1.5 text-sm font-medium ${ahead ? "bg-emerald-400/15 text-emerald-300" : "bg-rose-400/15 text-rose-300"}`}>
                {ahead ? "выше" : "ниже"} темпа на {rub(Math.abs(progress.paceDelta))}
              </div>
            ) : (
              <div className="rounded-full bg-white/10 px-3 py-1.5 text-sm font-medium text-slate-300">нет факта за месяц</div>
            )}
          </div>

          <div className="mt-7">
            <div className="h-3 overflow-hidden rounded-full bg-white/10">
              <div
                className="h-full rounded-full bg-indigo-400 transition-[width] duration-700 ease-out"
                style={{ width: `${progressWidth}%` }}
              />
            </div>
            <div className="mt-2 flex items-center justify-between text-sm">
              <span className="font-medium text-white">{pct(progress.completionPct)} выполнено</span>
              <span className="text-slate-400">{progress.dataThrough ? `план к этой дате: ${rub(progress.planToDate)}` : "темп появится вместе с фактом"}</span>
            </div>
          </div>

          <div className="mt-7 grid gap-5 border-t border-white/10 pt-5 sm:grid-cols-3">
            <div>
              <p className="text-xs uppercase tracking-wide text-slate-400">Прогноз месяца</p>
              <p className="mt-1 text-2xl font-semibold tabular-nums">{valueOrDash(progress.forecastNet, rub)}</p>
              {progress.forecastNet != null && (
                <p className={`mt-1 text-xs ${forecastOnTrack ? "text-emerald-300" : "text-rose-300"}`}>
                  {forecastOnTrack ? "цель достижима текущим темпом" : `не хватает ${rub(progress.target - progress.forecastNet)}`}
                </p>
              )}
            </div>
            <div>
              <p className="text-xs uppercase tracking-wide text-slate-400">Ожидаемый NET текущих заказов</p>
              <p className="mt-1 text-2xl font-semibold tabular-nums">{rub(progress.projectedNet)}</p>
              <p className="mt-1 text-xs text-slate-400">с учётом исторического G2N</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-wide text-slate-400">Осталось до цели</p>
              <p className="mt-1 text-2xl font-semibold tabular-nums">{rub(progress.remainingRevenue)}</p>
              <p className="mt-1 text-xs text-slate-400">{progress.dataThrough ? `${num(progress.remainingDays)} дн. после даты факта` : `${num(progress.remainingDays)} дн. в месяце`}</p>
            </div>
          </div>
        </div>
      </section>

      <section className="rounded-2xl bg-white p-5 ring-1 ring-slate-200 sm:p-7">
        <SourceNote status={analyticsStatus} dataThrough={progress.dataThrough} />
        {!progress.dataThrough && (
          <div className="mt-4 rounded-lg bg-amber-50 px-4 py-3 text-sm text-amber-800 ring-1 ring-amber-200">
            За выбранный месяц заказов пока нет. План сохранён, но оценить темп можно будет после появления факта.
          </div>
        )}
        {incompleteCoverage && (
          <div className="mt-4 rounded-lg bg-amber-50 px-4 py-3 text-sm text-amber-800 ring-1 ring-amber-200">
            Данные месяца начинаются с {shortDate(progress.coverageStarts)}. Прогноз может быть занижен из-за неполного начала периода.
          </div>
        )}

        <div className="mt-6 grid gap-7 lg:grid-cols-[0.9fr_1.1fr]">
          <div>
            <h2 className="text-base font-semibold text-ink">Чтобы выполнить цель</h2>
            <p className="mt-1 text-sm text-slate-500">Необходимый ежедневный темп на оставшуюся часть месяца.</p>
            <div className="mt-5 divide-y divide-slate-100 border-y border-slate-100">
              <div className="flex items-baseline justify-between gap-3 py-4">
                <span className="text-sm text-slate-500">NET в день</span>
                <strong className="text-xl tabular-nums text-ink">{valueOrDash(progress.requiredRevenuePerDay, rub)}</strong>
              </div>
              <div className="flex items-baseline justify-between gap-3 py-4">
                <span className="text-sm text-slate-500">Заказов в день</span>
                <strong className="text-xl tabular-nums text-ink">{valueOrDash(progress.requiredOrdersPerDay, num)}</strong>
              </div>
              <div className="flex items-baseline justify-between gap-3 py-4">
                <span className="text-sm text-slate-500">Визитов / сессий в день</span>
                <strong className="text-xl tabular-nums text-ink">{valueOrDash(progress.requiredSessionsPerDay, num)}</strong>
              </div>
            </div>
            {progress.requiredSessionsPerDay == null && (
              <p className="mt-3 text-xs text-amber-700">Для расчёта необходимых визитов нужны данные Метрики и AppMetrica.</p>
            )}
          </div>

          <div>
            <h2 className="text-base font-semibold text-ink">Фактические рычаги</h2>
            <p className="mt-1 text-sm text-slate-500">
              {progress.driver.source === "current"
                ? "Трафик и конверсия выбранного месяца; G2N — из зрелой истории."
                : progress.driver.source === "history"
                  ? "В месяце ещё мало данных — используем последние доступные 30 дней."
                  : "Трафик не загружен — денежные показатели считаются только по заказам."}
            </p>
            <div className="mt-5 grid grid-cols-2 gap-x-6 gap-y-5 sm:grid-cols-4 lg:grid-cols-2 xl:grid-cols-4">
              <div>
                <p className="text-xs uppercase tracking-wide text-slate-400">Трафик</p>
                <p className="mt-1 text-xl font-semibold tabular-nums text-ink">{num(progress.driver.sessions)}</p>
              </div>
              <div>
                <p className="text-xs uppercase tracking-wide text-slate-400">CR в заказ</p>
                <p className="mt-1 text-xl font-semibold tabular-nums text-ink">{progress.driver.netCr > 0 ? pct2(progress.driver.netCr * 100) : "—"}</p>
              </div>
              <div>
                <p className="text-xs uppercase tracking-wide text-slate-400">AOV</p>
                <p className="mt-1 text-xl font-semibold tabular-nums text-ink">{progress.driver.aov > 0 ? rub(progress.driver.aov) : "—"}</p>
              </div>
              <div>
                <p className="text-xs uppercase tracking-wide text-slate-400">G2N</p>
                <p className="mt-1 text-xl font-semibold tabular-nums text-ink">{progress.driver.g2n > 0 ? pct(progress.driver.g2n * 100) : "—"}</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="rounded-2xl bg-white p-5 ring-1 ring-slate-200 sm:p-7">
        <div className="flex flex-wrap items-end justify-between gap-2">
          <div>
            <h2 className="text-base font-semibold text-ink">Сайт и приложение</h2>
            <p className="mt-1 text-sm text-slate-500">
              Цель распределена по структуре продаж: сайт {pct(allocation.siteShare * 100)}, приложение {pct(allocation.appShare * 100)}.
            </p>
          </div>
          {allocation.source === "history" && (
            <span className="text-xs text-slate-400">База: {shortDate(allocation.start)} — {shortDate(allocation.end)}</span>
          )}
        </div>

        <div className="mt-5 divide-y divide-slate-100 border-y border-slate-100">
          {progress.channels.map((item) => (
            <div key={item.channel} className="grid gap-4 py-5 sm:grid-cols-[1fr_repeat(4,minmax(0,0.75fr))] sm:items-center">
              <div>
                <div className="flex items-center gap-2">
                  <span className={`h-2.5 w-2.5 rounded-full ${item.channel === "site" ? "bg-sky-500" : "bg-violet-500"}`} />
                  <h3 className="font-semibold text-ink">{item.label}</h3>
                </div>
                <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-slate-100">
                  <div
                    className={`h-full rounded-full transition-[width] duration-700 ${item.channel === "site" ? "bg-sky-500" : "bg-violet-500"}`}
                    style={{ width: `${Math.min(100, item.completionPct)}%` }}
                  />
                </div>
              </div>
              <div>
                <p className="text-xs text-slate-400">План</p>
                <p className="mt-0.5 font-semibold tabular-nums text-ink">{rub(item.target)}</p>
              </div>
              <div>
                <p className="text-xs text-slate-400">Факт NET</p>
                <p className="mt-0.5 font-semibold tabular-nums text-ink">{rub(item.factNet)}</p>
                <p className="text-[11px] text-slate-400">{pct(item.completionPct)} плана</p>
              </div>
              <div>
                <p className="text-xs text-slate-400">Трафик</p>
                <p className="mt-0.5 font-semibold tabular-nums text-ink">{num(item.sessions)}</p>
                <p className="text-[11px] text-slate-400">CR {item.netCr > 0 ? pct2(item.netCr) : "—"}</p>
              </div>
              <div>
                <p className="text-xs text-slate-400">Прогноз</p>
                <p className="mt-0.5 font-semibold tabular-nums text-ink">{valueOrDash(item.forecastNet, rub)}</p>
                <p className="text-[11px] text-slate-400">доля факта {pct(item.factShare * 100)}</p>
              </div>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
