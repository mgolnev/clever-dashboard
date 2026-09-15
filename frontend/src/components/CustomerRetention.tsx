import { useMemo, useState } from "react";
import type {
  CustomerAnalyticsReport,
  CustomerCohort,
  CustomerCompositionPeriod,
  CustomerGranularity,
  CustomerRetentionView,
  RetentionPoint,
} from "../types";
import { num, pct } from "../utils/format";

type Mode = "gross" | "paid";

interface Props {
  report: CustomerAnalyticsReport;
  onGranularityChange: (granularity: CustomerGranularity) => void;
}

const monthFormatter = new Intl.DateTimeFormat("ru-RU", { month: "short", year: "numeric" });

function periodLabel(value: string, granularity: CustomerGranularity): string {
  if (granularity === "quarter") {
    const date = new Date(`${value.slice(0, 10)}T00:00:00`);
    return `${Math.floor(date.getMonth() / 3) + 1} кв. ${date.getFullYear()}`;
  }
  return monthFormatter.format(new Date(`${value.slice(0, 10)}T00:00:00`)).replace(" г.", "");
}

function pointAt(view: CustomerRetentionView, offset: number): RetentionPoint | undefined {
  return view.summary.retention.find((point) => point.offset === offset && point.eligibleCustomers > 0);
}

function metricValue(point?: RetentionPoint): string {
  return point ? pct(point.rate) : "—";
}

function strongestM1(cohorts: CustomerCohort[]): CustomerCohort | undefined {
  return cohorts
    .filter((cohort) => cohort.cells[1]?.complete)
    .sort((a, b) => (b.cells[1]?.rate ?? 0) - (a.cells[1]?.rate ?? 0))[0];
}

function denominatorExample(view: CustomerRetentionView, granularity: CustomerGranularity): string | null {
  for (const cohort of view.cohorts) {
    const nextPeriod = cohort.cells[1];
    if (!nextPeriod?.complete || nextPeriod.customers === 0) continue;
    const composition = view.composition.find((period) => period.period === nextPeriod.month && period.complete);
    if (!composition || composition.active !== nextPeriod.customers) continue;
    return `${num(nextPeriod.customers)} / ${num(cohort.customers)} = ${pct(nextPeriod.rate)} retention когорты ${periodLabel(cohort.month, granularity)}; `
      + `${num(composition.active)} / ${num(composition.customers)} = ${pct(composition.activeRate)} доли активных в ${periodLabel(composition.period, granularity)}.`;
  }
  return null;
}

function heatStyle(rate: number): { backgroundColor: string; color: string } {
  if (rate <= 0) return { backgroundColor: "#f8fafc", color: "#94a3b8" };
  const intensity = Math.min(0.92, 0.12 + rate / 115);
  return {
    backgroundColor: `rgba(79, 70, 229, ${intensity})`,
    color: intensity > 0.52 ? "#ffffff" : "#3730a3",
  };
}

export default function CustomerRetention({ report, onGranularityChange }: Props) {
  const [mode, setMode] = useState<Mode>("gross");
  const granularity = report.granularity;
  const offsetPrefix = granularity === "month" ? "M" : "Q";
  const periodNoun = granularity === "month" ? "месяц" : "квартал";
  const longOffset = granularity === "month" ? 3 : 2;
  const view = report[mode];
  const m1 = pointAt(view, 1);
  const m2 = pointAt(view, 2);
  const longRetention = pointAt(view, longOffset);
  const grossM1 = pointAt(report.gross, 1);
  const paidM1 = pointAt(report.paid, 1);
  const best = useMemo(() => strongestM1(view.cohorts), [view.cohorts]);
  const maxOffset = Math.max(0, ...view.cohorts.flatMap((cohort) => cohort.cells.map((cell) => cell.offset)));
  const offsets = Array.from({ length: maxOffset + 1 }, (_, index) => index);
  const paidGap = grossM1 && paidM1 ? paidM1.rate - grossM1.rate : null;
  const latestCompleteComposition = [...view.composition].reverse().find((period) => period.complete);
  const compositionDenominatorExample = denominatorExample(view, granularity);

  return (
    <section className="overflow-hidden rounded-xl bg-white shadow-sm ring-1 ring-slate-200">
      <div className="flex flex-col gap-5 border-b border-slate-100 px-5 py-5 lg:flex-row lg:items-start lg:justify-between">
        <div className="max-w-2xl">
          <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-brand">Когортный анализ</p>
          <h2 className="mt-1 text-xl font-semibold tracking-tight text-ink">Возвращаются ли новые клиенты</h2>
          <p className="mt-1.5 text-sm leading-6 text-slate-500">
            Когорта — {periodNoun} первого {mode === "gross" ? "оформленного" : "оплаченного"} заказа.
            {" "}
            {offsetPrefix}1 показывает возврат в следующий календарный {periodNoun}, {offsetPrefix}2 — через два.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <SegmentedControl
            label="Режим заказов"
            items={[
              { value: "gross", label: "Гросс" },
              { value: "paid", label: "Оплаченные" },
            ]}
            value={mode}
            onChange={(value) => setMode(value as Mode)}
          />
          <SegmentedControl
            label="Период анализа"
            items={[
              { value: "month", label: "Месяц" },
              { value: "quarter", label: "Квартал" },
            ]}
            value={granularity}
            onChange={(value) => onGranularityChange(value as CustomerGranularity)}
          />
        </div>
      </div>

      <div className="grid grid-cols-2 border-b border-slate-100 lg:grid-cols-4">
        {[
          { label: mode === "gross" ? "Новых клиентов" : "Платящих клиентов", value: num(view.summary.customers), hint: `${view.summary.cohortCount} когорт` },
          { label: "Вернулись позже", value: pct(view.summary.repeatRate), hint: `${num(view.summary.repeatCustomers)} клиентов` },
          { label: `Retention ${offsetPrefix}1`, value: metricValue(m1), hint: m1 ? `база ${num(m1.eligibleCustomers)}` : "нет зрелых когорт" },
          { label: `Retention ${offsetPrefix}${longOffset}`, value: metricValue(longRetention), hint: longRetention ? `база ${num(longRetention.eligibleCustomers)}` : "нет зрелых когорт" },
        ].map((item, index) => (
          <div
            key={item.label}
            className={`px-5 py-4 ${index % 2 === 0 ? "border-r" : ""} border-slate-100 lg:border-r lg:last:border-r-0`}
          >
            <div className="text-xs font-medium text-slate-500">{item.label}</div>
            <div className="mt-1 text-2xl font-semibold tabular-nums tracking-tight text-ink">{item.value}</div>
            <div className="mt-0.5 text-xs text-slate-400">{item.hint}</div>
          </div>
        ))}
      </div>

      <div className="grid border-b border-slate-100 lg:grid-cols-3">
        <Insight
          eyebrow="Первый повтор"
          title={m1 ? `${pct(m1.rate)} возвращаются в ${offsetPrefix}1` : `${offsetPrefix}1 ещё не сформирован`}
          body={m2 ? `К ${offsetPrefix}2 активны ${pct(m2.rate)} клиентов из уже зрелых когорт.` : `Для ${offsetPrefix}2 пока нет достаточно зрелых когорт.`}
        />
        <Insight
          eyebrow="Оплата vs гросс"
          title={paidGap === null ? "Пока недостаточно данных" : `${paidGap >= 0 ? "+" : ""}${paidGap.toFixed(1).replace(".", ",")} п.п. в ${offsetPrefix}1`}
          body={paidGap === null ? "Сравнение появится после созревания когорт." : paidGap >= 0 ? "Оплаченные клиенты возвращаются чаще, чем все оформившие заказ." : "Гросс-повтор выше повтора с подтверждённой оплатой."}
        />
        <Insight
          eyebrow="Лучшая когорта"
          title={best ? `${periodLabel(best.month, granularity)} · ${pct(best.cells[1].rate)}` : "Нужна зрелая когорта"}
          body={best ? `${num(best.cells[1].customers)} из ${num(best.customers)} клиентов вернулись в следующий ${periodNoun}.` : `Текущий ${periodNoun} не сравниваем с завершёнными.`}
        />
      </div>

      <div className="px-5 py-5">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <div>
            <h3 className="text-sm font-semibold text-ink">Матрица retention</h3>
            <p className="mt-0.5 text-xs text-slate-400">
              {periodLabel(report.period.start, granularity)} — {periodLabel(report.period.end, granularity)} · неполные периоды отмечены * и не входят в KPI
            </p>
          </div>
          <div className="flex items-center gap-3 text-[11px] text-slate-400">
            <span className="flex items-center gap-1.5"><i className="h-2.5 w-2.5 rounded-sm bg-slate-100" />0%</span>
            <span className="flex items-center gap-1.5"><i className="h-2.5 w-2.5 rounded-sm bg-indigo-300" />50%</span>
            <span className="flex items-center gap-1.5"><i className="h-2.5 w-2.5 rounded-sm bg-brand" />100%</span>
          </div>
        </div>

        {view.cohorts.length === 0 ? (
          <div className="py-10 text-center text-sm text-slate-400">Для выбранных фильтров нет когорт</div>
        ) : (
          <div>
            <div className="relative">
              <div className="overflow-x-auto pb-1 overscroll-x-contain">
                <table key={mode} className="retention-enter w-full min-w-[720px] border-separate border-spacing-1 text-sm">
              <thead>
                <tr className="text-[11px] uppercase tracking-wide text-slate-400">
                  <th className="sticky left-0 z-10 w-32 bg-white pb-1 pr-3 text-left font-medium">Когорта</th>
                  <th className="w-20 pb-1 text-right font-medium">Клиенты</th>
                  {offsets.map((offset) => (
                    <th key={offset} className="min-w-[66px] pb-1 text-center font-medium">{offsetPrefix}{offset}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {view.cohorts.map((cohort) => (
                  <tr key={cohort.month}>
                    <th className="sticky left-0 z-10 bg-white py-1 pr-3 text-left font-medium capitalize text-slate-700">
                      {periodLabel(cohort.month, granularity)}
                    </th>
                    <td className="pr-2 text-right tabular-nums text-slate-500">{num(cohort.customers)}</td>
                    {cohort.cells.map((cell) => (
                      <td key={cell.offset}>
                        {cell.available ? (
                          <div
                            title={`${periodLabel(cohort.month, granularity)}, ${offsetPrefix}${cell.offset}: ${num(cell.customers)} из ${num(cohort.customers)} (${pct(cell.rate)})${cell.complete ? "" : " · неполный период наблюдения"}`}
                            className={`flex h-10 min-w-[62px] items-center justify-center rounded-md text-xs font-semibold tabular-nums transition-transform duration-150 hover:scale-[1.04] hover:ring-2 hover:ring-brand/20 ${cell.complete ? "" : "border border-dashed border-slate-400"}`}
                            style={heatStyle(cell.rate)}
                          >
                            {pct(cell.rate)}{!cell.complete ? "*" : ""}
                          </div>
                        ) : (
                          <div className="flex h-10 min-w-[62px] items-center justify-center text-slate-200">—</div>
                        )}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
              <tfoot>
                <tr>
                  <th className="sticky left-0 z-10 bg-white pt-2 pr-3 text-left text-xs font-semibold text-slate-500">Все зрелые</th>
                  <td />
                  {offsets.map((offset) => {
                    const point = pointAt(view, offset);
                    return (
                      <td key={offset} className="pt-2 text-center text-xs font-semibold tabular-nums text-brand">
                        {point ? pct(point.rate) : "—"}
                      </td>
                    );
                  })}
                </tr>
              </tfoot>
                </table>
              </div>
              <div className="pointer-events-none absolute inset-y-0 right-0 w-6 bg-gradient-to-l from-white to-transparent sm:hidden" aria-hidden="true" />
            </div>
            <MobileTableScrollHint />
          </div>
        )}
      </div>

      <div className="border-t border-slate-100 px-5 py-5">
        <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h3 className="text-sm font-semibold text-ink">Состав клиентской базы</h3>
            <p className="mt-1 max-w-3xl text-xs leading-5 text-slate-500">
              Новые делают первый заказ; активные заказывали и в предыдущем периоде; возвращённые пропустили предыдущий период, но покупали раньше.
              Проценты в этой таблице считаются от всех клиентов соответствующего периода.
            </p>
            {compositionDenominatorExample && (
              <p className="mt-1 text-xs leading-5 text-slate-400">
                Разные базы расчёта: {compositionDenominatorExample}
              </p>
            )}
          </div>
          {latestCompleteComposition && (
            <p className="text-xs text-slate-400">
              Последний полный: <span className="font-medium text-slate-600">{periodLabel(latestCompleteComposition.period, granularity)}</span>
            </p>
          )}
        </div>
        <CompositionTable rows={view.composition} granularity={granularity} />
      </div>
    </section>
  );
}

function SegmentedControl({
  label,
  items,
  value,
  onChange,
}: {
  label: string;
  items: { value: string; label: string }[];
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="inline-flex w-fit rounded-lg bg-slate-100 p-1" aria-label={label}>
      {items.map((item) => (
        <button
          key={item.value}
          type="button"
          onClick={() => onChange(item.value)}
          className={`rounded-md px-3 py-1.5 text-sm font-medium transition-all duration-200 ${
            value === item.value ? "bg-white text-ink shadow-sm" : "text-slate-500 hover:text-slate-700"
          }`}
        >
          {item.label}
        </button>
      ))}
    </div>
  );
}

function CompositionTable({ rows, granularity }: { rows: CustomerCompositionPeriod[]; granularity: CustomerGranularity }) {
  if (rows.length === 0) {
    return <div className="py-8 text-center text-sm text-slate-400">Нет данных о составе клиентской базы</div>;
  }
  return (
    <div>
      <div className="relative">
        <div className="overflow-x-auto overscroll-x-contain">
          <table className="w-full min-w-[820px] text-sm">
        <thead>
          <tr className="border-b border-slate-100 text-[11px] uppercase tracking-wide text-slate-400">
            <th className="pb-2 text-left font-medium">Период</th>
            <th className="pb-2 text-right font-medium">Клиенты</th>
            <th className="w-[36%] px-5 pb-2 text-left font-medium">Структура</th>
            <th className="pb-2 text-right font-medium"><LegendDot className="bg-brand" />Новые</th>
            <th className="pb-2 text-right font-medium"><LegendDot className="bg-slate-700" />Активные</th>
            <th className="pb-2 text-right font-medium"><LegendDot className="bg-indigo-200" />Вернулись</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.period} className="border-b border-slate-50 last:border-0">
              <th className="py-3 text-left font-medium text-slate-700">
                {periodLabel(row.period, granularity)}{!row.complete ? <span className="ml-1 text-slate-400">*</span> : null}
              </th>
              <td className="py-3 text-right font-semibold tabular-nums text-ink">{num(row.customers)}</td>
              <td className="px-5 py-3">
                <div className="flex h-2.5 w-full overflow-hidden rounded-full bg-slate-100" title={`Новые ${pct(row.newRate)} · активные ${pct(row.activeRate)} · вернувшиеся ${pct(row.returnedRate)}`}>
                  <span className="bg-brand transition-[width] duration-300" style={{ width: `${row.newRate}%` }} />
                  <span className="bg-slate-700 transition-[width] duration-300" style={{ width: `${row.activeRate}%` }} />
                  <span className="bg-indigo-200 transition-[width] duration-300" style={{ width: `${row.returnedRate}%` }} />
                </div>
              </td>
              <CompositionValue value={row.new} rate={row.newRate} total={row.customers} />
              <CompositionValue value={row.active} rate={row.activeRate} total={row.customers} />
              <CompositionValue value={row.returned} rate={row.returnedRate} total={row.customers} />
            </tr>
          ))}
        </tbody>
          </table>
        </div>
        <div className="pointer-events-none absolute inset-y-0 right-0 w-6 bg-gradient-to-l from-white to-transparent sm:hidden" aria-hidden="true" />
      </div>
      <MobileTableScrollHint />
      <p className="mt-2 text-[11px] text-slate-400">* незавершённый период</p>
    </div>
  );
}

function MobileTableScrollHint() {
  return (
    <p className="mt-2 flex items-center justify-end gap-1.5 text-[11px] text-slate-400 sm:hidden">
      <span aria-hidden="true">←</span>
      Проведите по таблице
      <span aria-hidden="true">→</span>
    </p>
  );
}

function LegendDot({ className }: { className: string }) {
  return <i className={`mr-1.5 inline-block h-2 w-2 rounded-sm ${className}`} />;
}

function CompositionValue({ value, rate, total }: { value: number; rate: number; total: number }) {
  return (
    <td className="py-3 text-right tabular-nums text-slate-600" title={`${num(value)} / ${num(total)} клиентов периода = ${pct(rate)}`}>
      <span className="font-medium text-slate-700">{num(value)}</span>
      <span className="ml-1.5 text-xs text-slate-400">{pct(rate)}</span>
    </td>
  );
}

function Insight({ eyebrow, title, body }: { eyebrow: string; title: string; body: string }) {
  return (
    <div className="border-b border-slate-100 px-5 py-4 last:border-b-0 lg:border-b-0 lg:border-r lg:last:border-r-0">
      <div className="text-[10px] font-semibold uppercase tracking-[0.14em] text-slate-400">{eyebrow}</div>
      <div className="mt-1 text-sm font-semibold text-slate-700">{title}</div>
      <p className="mt-1 text-xs leading-5 text-slate-500">{body}</p>
    </div>
  );
}
