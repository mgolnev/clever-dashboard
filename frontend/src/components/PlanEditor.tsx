import { useEffect, useState } from "react";
import type { PlanReport, TrafficReport } from "../types";
import { api } from "../api";
import { num } from "../utils/format";

const MONTH_NAMES = [
  "Янв", "Фев", "Мар", "Апр", "Май", "Июн",
  "Июл", "Авг", "Сен", "Окт", "Ноя", "Дек",
];

interface Props {
  year: number;
  plan: PlanReport;
  traffic: TrafficReport;
  onSaved: (plan: PlanReport, traffic: TrafficReport) => void;
}

type Draft = {
  planAll: number[];
  planSite: number[];
  planApp: number[];
  trafficSite: number[];
  trafficApp: number[];
};

function fromReports(plan: PlanReport, traffic: TrafficReport): Draft {
  return {
    planAll: plan.months.map((m) => m.targets.all),
    planSite: plan.months.map((m) => m.targets.site),
    planApp: plan.months.map((m) => m.targets.app),
    trafficSite: traffic.months.map((m) => m.site),
    trafficApp: traffic.months.map((m) => m.app),
  };
}

function NumInput({
  value,
  onChange,
}: {
  value: number;
  onChange: (v: number) => void;
}) {
  return (
    <input
      type="number"
      min={0}
      step={1000}
      value={value || ""}
      onChange={(e) => onChange(Math.max(0, parseInt(e.target.value, 10) || 0))}
      className="w-full min-w-[4.5rem] rounded border border-slate-200 px-1.5 py-1 text-right text-xs focus:border-brand focus:outline-none focus:ring-1 focus:ring-brand"
    />
  );
}

export default function PlanEditor({ year, plan, traffic, onSaved }: Props) {
  const [draft, setDraft] = useState<Draft>(() => fromReports(plan, traffic));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setDraft(fromReports(plan, traffic));
  }, [plan, traffic]);

  const setPlan = (field: keyof Pick<Draft, "planAll" | "planSite" | "planApp">, idx: number, v: number) => {
    setDraft((d) => {
      const arr = [...d[field]];
      arr[idx] = v;
      return { ...d, [field]: arr };
    });
  };

  const setTraffic = (field: "trafficSite" | "trafficApp", idx: number, v: number) => {
    setDraft((d) => {
      const arr = [...d[field]];
      arr[idx] = v;
      return { ...d, [field]: arr };
    });
  };

  const save = async () => {
    setSaving(true);
    setError(null);
    try {
      const planItems: { month: number; channel: string; netTarget: number }[] = [];
      const trafficItems: { month: number; channel: string; visits: number }[] = [];
      for (let i = 0; i < 12; i++) {
        const month = i + 1;
        planItems.push(
          { month, channel: "all", netTarget: draft.planAll[i] },
          { month, channel: "site", netTarget: draft.planSite[i] },
          { month, channel: "app", netTarget: draft.planApp[i] }
        );
        trafficItems.push(
          { month, channel: "site", visits: draft.trafficSite[i] },
          { month, channel: "app", visits: draft.trafficApp[i] }
        );
      }
      const p = await api.putPlan(year, planItems);
      const t = await api.putTraffic(year, trafficItems);
      onSaved(p, t);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка сохранения");
    } finally {
      setSaving(false);
    }
  };

  const cols = [
    { key: "planAll" as const, label: "План NET (всего)", group: "План" },
    { key: "planSite" as const, label: "План NET (сайт)", group: "План" },
    { key: "planApp" as const, label: "План NET (прил.)", group: "План" },
    { key: "trafficSite" as const, label: "Трафик (сайт)", group: "Трафик" },
    { key: "trafficApp" as const, label: "Трафик (прил.)", group: "Трафик" },
  ];

  return (
    <div className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-base font-semibold text-ink">План и трафик на {year} год</h2>
        <button
          type="button"
          onClick={save}
          disabled={saving}
          className="rounded-lg bg-brand px-4 py-2 text-sm font-medium text-white transition hover:bg-brand/90 disabled:opacity-50"
        >
          {saving ? "Сохранение…" : "Сохранить"}
        </button>
      </div>
      {error && <p className="mb-3 text-sm text-rose-600">{error}</p>}
      <div className="overflow-x-auto">
        <table className="w-full min-w-[640px] text-left text-xs">
          <thead>
            <tr className="border-b border-slate-200 text-slate-500">
              <th className="py-2 pr-2 font-medium">Месяц</th>
              {cols.map((c) => (
                <th key={c.key} className="px-1 py-2 font-medium">
                  <span className="block text-[10px] uppercase text-slate-400">{c.group}</span>
                  {c.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {MONTH_NAMES.map((name, i) => (
              <tr key={name} className="border-b border-slate-100">
                <td className="py-1.5 pr-2 font-medium text-ink">
                  {name}
                  <span className="ml-1 text-slate-400">({num(plan.months[i].daysInMonth)} д.)</span>
                </td>
                <td className="px-1 py-1">
                  <NumInput value={draft.planAll[i]} onChange={(v) => setPlan("planAll", i, v)} />
                </td>
                <td className="px-1 py-1">
                  <NumInput value={draft.planSite[i]} onChange={(v) => setPlan("planSite", i, v)} />
                </td>
                <td className="px-1 py-1">
                  <NumInput value={draft.planApp[i]} onChange={(v) => setPlan("planApp", i, v)} />
                </td>
                <td className="px-1 py-1">
                  <NumInput value={draft.trafficSite[i]} onChange={(v) => setTraffic("trafficSite", i, v)} />
                </td>
                <td className="px-1 py-1">
                  <NumInput value={draft.trafficApp[i]} onChange={(v) => setTraffic("trafficApp", i, v)} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
