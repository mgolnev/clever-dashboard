import type { City, Range } from "../types";
import { num } from "../utils/format";

interface Props {
  start: string;
  end: string;
  min?: string;
  max?: string;
  previous?: Range;
  cities: City[];
  city: string;
  onCityChange: (city: string) => void;
  regions: City[];
  region: string;
  onRegionChange: (region: string) => void;
  onChange: (start: string, end: string) => void;
}

function addDays(date: string, days: number): string {
  const d = new Date(date + "T00:00:00");
  d.setDate(d.getDate() + days);
  return d.toISOString().slice(0, 10);
}

export default function DateRangeBar({
  start,
  end,
  min,
  max,
  previous,
  cities,
  city,
  onCityChange,
  regions,
  region,
  onRegionChange,
  onChange,
}: Props) {
  const anchor = max || end;

  const presets: { label: string; range: () => [string, string] }[] = [
    { label: "7 дней", range: () => [addDays(anchor, -6), anchor] },
    { label: "14 дней", range: () => [addDays(anchor, -13), anchor] },
    { label: "30 дней", range: () => [addDays(anchor, -29), anchor] },
    {
      label: "Текущий месяц",
      range: () => {
        const d = new Date(anchor + "T00:00:00");
        const first = new Date(d.getFullYear(), d.getMonth(), 1);
        return [first.toISOString().slice(0, 10), anchor];
      },
    },
  ];

  return (
    <div className="flex flex-wrap items-end gap-3 rounded-xl bg-white p-4 shadow-sm ring-1 ring-slate-200">
      <label className="flex flex-col text-xs font-medium text-slate-500">
        Начало
        <input
          type="date"
          value={start}
          min={min}
          max={max}
          onChange={(e) => onChange(e.target.value, end)}
          className="mt-1 rounded-lg border border-slate-300 px-3 py-1.5 text-sm text-ink"
        />
      </label>
      <label className="flex flex-col text-xs font-medium text-slate-500">
        Конец
        <input
          type="date"
          value={end}
          min={min}
          max={max}
          onChange={(e) => onChange(start, e.target.value)}
          className="mt-1 rounded-lg border border-slate-300 px-3 py-1.5 text-sm text-ink"
        />
      </label>
      <label className="flex flex-col text-xs font-medium text-slate-500">
        Область
        <select
          value={region}
          onChange={(e) => onRegionChange(e.target.value)}
          className="mt-1 max-w-[200px] rounded-lg border border-slate-300 px-3 py-1.5 text-sm text-ink"
        >
          <option value="">Все области</option>
          {regions.map((r) => (
            <option key={r.name} value={r.name}>
              {r.name} ({num(r.orders)})
            </option>
          ))}
        </select>
      </label>
      <label className="flex flex-col text-xs font-medium text-slate-500">
        Город
        <select
          value={city}
          onChange={(e) => onCityChange(e.target.value)}
          className="mt-1 max-w-[180px] rounded-lg border border-slate-300 px-3 py-1.5 text-sm text-ink"
        >
          <option value="">Все города</option>
          {cities.map((c) => (
            <option key={c.name} value={c.name}>
              {c.name} ({num(c.orders)})
            </option>
          ))}
        </select>
      </label>
      <div className="flex flex-wrap gap-1.5">
        {presets.map((p) => (
          <button
            key={p.label}
            onClick={() => {
              const [s, e] = p.range();
              onChange(s, e);
            }}
            className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm text-slate-600 transition hover:border-brand hover:text-brand"
          >
            {p.label}
          </button>
        ))}
      </div>
      {previous && (
        <div className="ml-auto text-xs text-slate-400">
          Сравнение с периодом: <span className="font-medium text-slate-500">{previous.start} — {previous.end}</span>
        </div>
      )}
    </div>
  );
}
