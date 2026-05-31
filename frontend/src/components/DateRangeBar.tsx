import type { City, Range } from "../types";
import MultiSelect from "./MultiSelect";

interface Props {
  start: string;
  end: string;
  min?: string;
  max?: string;
  previous?: Range;
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
      {previous && (
        <div className="ml-auto text-xs text-slate-400">
          Сравнение с периодом: <span className="font-medium text-slate-500">{previous.start} — {previous.end}</span>
        </div>
      )}
    </div>
  );
}
