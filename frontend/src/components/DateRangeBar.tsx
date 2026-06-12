import type { City, Range } from "../types";
import MultiSelect from "./MultiSelect";

type CompareMode = "prev" | "yoy" | "prevMonth" | "custom";

interface Props {
  start: string;
  end: string;
  min?: string;
  max?: string;
  previous?: Range;
  compareEnabled: boolean;
  onToggleCompare: (enabled: boolean) => void;
  compareMode: CompareMode;
  onCompareModeChange: (mode: CompareMode) => void;
  compareStart: string;
  compareEnd: string;
  onCompareRangeChange: (start: string, end: string) => void;
  cities: City[];
  city: string[];
  onCityChange: (city: string[]) => void;
  regions: City[];
  region: string[];
  onRegionChange: (region: string[]) => void;
  channels: City[];
  channel: string[];
  onChannelChange: (channel: string[]) => void;
  payments: City[];
  payment: string[];
  onPaymentChange: (payment: string[]) => void;
  deliveries: City[];
  delivery: string[];
  onDeliveryChange: (delivery: string[]) => void;
  coupons: City[];
  coupon: string[];
  onCouponChange: (coupon: string[]) => void;
  onChange: (start: string, end: string) => void;
}

const COMPARE_MODES: { key: CompareMode; label: string }[] = [
  { key: "prev", label: "Предыдущий период" },
  { key: "yoy", label: "Год назад" },
  { key: "prevMonth", label: "Месяц назад" },
  { key: "custom", label: "Произвольный" },
];

export default function DateRangeBar({
  start,
  end,
  min,
  max,
  previous,
  compareEnabled,
  onToggleCompare,
  compareMode,
  onCompareModeChange,
  compareStart,
  compareEnd,
  onCompareRangeChange,
  cities,
  city,
  onCityChange,
  regions,
  region,
  onRegionChange,
  channels,
  channel,
  onChannelChange,
  payments,
  payment,
  onPaymentChange,
  deliveries,
  delivery,
  onDeliveryChange,
  coupons,
  coupon,
  onCouponChange,
  onChange,
}: Props) {
  return (
    <div className="flex flex-col gap-4 rounded-xl bg-white p-4 shadow-sm ring-1 ring-slate-200">
      {/* Первая строка: даты начала/конца и режим сравнения */}
      <div className="flex flex-wrap items-end gap-4 w-full">
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

        <div className="flex flex-col gap-3 border-l border-slate-200 pl-3">
          <div className="flex flex-wrap items-center gap-3">
            <label className="flex cursor-pointer items-center gap-2 text-xs font-medium text-slate-600 shrink-0">
              <input
                type="checkbox"
                checked={compareEnabled}
                onChange={(e) => onToggleCompare(e.target.checked)}
                className="h-3.5 w-3.5 rounded border-slate-300 text-brand focus:ring-brand"
              />
              Режим сравнения
            </label>
            {compareEnabled && (
              <div className="flex flex-wrap gap-1">
                {COMPARE_MODES.map((m) => (
                  <button
                    key={m.key}
                    type="button"
                    onClick={() => onCompareModeChange(m.key)}
                    className={`rounded-md px-2 py-0.5 text-[11px] font-medium transition ${
                      compareMode === m.key
                        ? "bg-indigo-100 text-indigo-800"
                        : "bg-slate-100 text-slate-600 hover:bg-slate-200"
                    }`}
                  >
                    {m.label}
                  </button>
                ))}
              </div>
            )}
          </div>
          {compareEnabled && compareMode === "custom" && (
            <div className="flex flex-wrap items-end gap-3">
              <label className="flex flex-col text-xs font-medium text-slate-500">
                Сравн. начало
                <input
                  type="date"
                  value={compareStart}
                  min={min}
                  max={max}
                  onChange={(e) => onCompareRangeChange(e.target.value, compareEnd)}
                  className="mt-1 rounded-lg border border-slate-300 px-3 py-1.5 text-sm text-ink"
                />
              </label>
              <label className="flex flex-col text-xs font-medium text-slate-500">
                Сравн. конец
                <input
                  type="date"
                  value={compareEnd}
                  min={min}
                  max={max}
                  onChange={(e) => onCompareRangeChange(compareStart, e.target.value)}
                  className="mt-1 rounded-lg border border-slate-300 px-3 py-1.5 text-sm text-ink"
                />
              </label>
            </div>
          )}
        </div>

        {compareEnabled && previous && (
          <div className="ml-auto text-xs text-slate-400 self-center">
            Сравнение с периодом:{" "}
            <span className="font-medium text-slate-500">
              {previous.start} — {previous.end}
            </span>
          </div>
        )}
      </div>

      {/* Вторая строка: остальные фильтры */}
      <div className="flex flex-wrap items-end gap-3 w-full border-t border-slate-100 pt-3">
        <MultiSelect
          label="Область"
          allLabel="Все области"
          options={regions}
          selected={region}
          onChange={onRegionChange}
          width={200}
        />
        <MultiSelect
          label="Город"
          allLabel="Все города"
          options={cities}
          selected={city}
          onChange={onCityChange}
          width={180}
        />
        <MultiSelect
          label="Витрина"
          allLabel="Все витрины"
          options={channels}
          selected={channel}
          onChange={onChannelChange}
          width={170}
        />
        <MultiSelect
          label="Способ оплаты"
          allLabel="Все способы"
          options={payments}
          selected={payment}
          onChange={onPaymentChange}
          width={200}
        />
        <MultiSelect
          label="Способ доставки"
          allLabel="Все способы"
          options={deliveries}
          selected={delivery}
          onChange={onDeliveryChange}
          width={220}
        />
        <MultiSelect
          label="Промокод"
          allLabel="Все промокоды"
          options={coupons}
          selected={coupon}
          onChange={onCouponChange}
          width={180}
        />
      </div>
    </div>
  );
}
