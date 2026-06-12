import type { ChannelGoal } from "../utils/planCalc";
import { num, pct, rub } from "../utils/format";

interface Props {
  goals: ChannelGoal[];
}

function MetricCard({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="rounded-xl bg-white p-4 shadow-sm ring-1 ring-slate-200">
      <p className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</p>
      <p className="mt-1 text-xl font-semibold text-ink">{value}</p>
      {hint && <p className="mt-0.5 text-xs text-slate-400">{hint}</p>}
    </div>
  );
}

function ChannelBlock({ goal }: { goal: ChannelGoal }) {
  const { metrics: m } = goal;
  const gapText =
    goal.gap > 0 ? rub(goal.gap) + " до плана" : goal.gap < 0 ? rub(-goal.gap) + " сверх" : "выполнен";

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-ink">{goal.label}</h3>
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        <MetricCard label="План NET" value={rub(goal.planTarget)} />
        <MetricCard label="Факт NET" value={rub(goal.factNet)} hint="выкупленная выручка" />
        <MetricCard label="Разрыв" value={gapText} />
        <MetricCard
          label="CR"
          value={m.visits > 0 ? pct(m.cr * 100) : "—"}
          hint={m.visits > 0 ? `${num(m.orders)} зак. / ${num(m.visits)} виз.` : "введите трафик"}
        />
        <MetricCard
          label="AOV"
          value={m.orders > 0 ? rub(Math.round(m.aov)) : "—"}
          hint="средний чек на заказ"
        />
        <MetricCard
          label="Выкупаемость R"
          value={m.revenue > 0 ? pct(m.r * 100) : "—"}
          hint="NET / выручка оформленных"
        />
        <MetricCard
          label="Нужно визитов"
          value={
            goal.requiredVisits != null
              ? num(goal.requiredVisits)
              : goal.planTarget > 0
                ? "нет данных"
                : "—"
          }
          hint={
            goal.requiredVisitsPerDay != null
              ? `~${num(goal.requiredVisitsPerDay)} в день на месяц`
              : undefined
          }
        />
        <MetricCard label="Факт визитов" value={num(goal.factVisits)} />
        <MetricCard
          label="Визитов/день, чтобы догнать"
          value={
            goal.catchUpVisitsPerDay != null
              ? num(goal.catchUpVisitsPerDay)
              : goal.requiredVisits != null && goal.factVisits >= goal.requiredVisits
                ? "—"
                : goal.planTarget > 0 && goal.requiredVisits == null
                  ? "нет данных"
                  : "—"
          }
          hint="остаток месяца"
        />
      </div>
    </div>
  );
}

export default function GoalSummary({ goals }: Props) {
  return (
    <div className="rounded-xl bg-slate-50 p-5 shadow-sm ring-1 ring-slate-200">
      <h2 className="mb-4 text-base font-semibold text-ink">Достижение плана</h2>
      <div className="space-y-6">
        {goals.map((g) => (
          <ChannelBlock key={g.channel} goal={g} />
        ))}
      </div>
    </div>
  );
}
