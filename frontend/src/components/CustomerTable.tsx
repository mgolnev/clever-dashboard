import { useMemo, useState } from "react";
import type { CustomerRow } from "../types";
import { num, pct, rub } from "../utils/format";

interface Props {
  rows: CustomerRow[];
  totalRevenue: number;
}

type SortKey =
  | "revenue"
  | "revenueShare"
  | "orders"
  | "aov"
  | "paidOrders"
  | "inTransitOrders"
  | "completedOrders"
  | "canceledOrders"
  | "g2n"
  | "p2n";

const COLUMNS: { key: SortKey; label: string; title?: string }[] = [
  { key: "revenue", label: "Выручка" },
  { key: "revenueShare", label: "Доля", title: "Доля от общей выручки периода" },
  { key: "orders", label: "Заказы", title: "Оформлено за период" },
  { key: "aov", label: "AOV", title: "Средний чек на заказ" },
  { key: "paidOrders", label: "Опл.", title: "Оплаченные заказы" },
  { key: "inTransitOrders", label: "Транзит", title: "Заказы в пути (не финальный статус)" },
  { key: "completedOrders", label: "Выкуп", title: "Выкупленные заказы" },
  { key: "canceledOrders", label: "Отм.", title: "Отменённые заказы" },
  { key: "g2n", label: "G2N", title: "Выкуп / оформлено" },
  { key: "p2n", label: "P2N", title: "Выкуп / оплачено" },
];

function aov(row: CustomerRow): number {
  return row.orders > 0 ? row.revenue / row.orders : 0;
}

function g2n(row: CustomerRow): number {
  return row.orders > 0 ? (row.completedOrders / row.orders) * 100 : 0;
}

function p2n(row: CustomerRow): number {
  return row.paidOrders > 0 ? (row.completedOrders / row.paidOrders) * 100 : 0;
}

function sortValue(row: CustomerRow, key: SortKey): number {
  switch (key) {
    case "revenue":
      return row.revenue;
    case "revenueShare":
      return row.revenueShare;
    case "orders":
      return row.orders;
    case "aov":
      return aov(row);
    case "paidOrders":
      return row.paidOrders;
    case "inTransitOrders":
      return row.inTransitOrders;
    case "completedOrders":
      return row.completedOrders;
    case "canceledOrders":
      return row.canceledOrders;
    case "g2n":
      return g2n(row);
    case "p2n":
      return p2n(row);
  }
}

function cellValue(row: CustomerRow, key: SortKey): string {
  switch (key) {
    case "revenue":
      return rub(row.revenue);
    case "revenueShare":
      return pct(row.revenueShare);
    case "orders":
      return num(row.orders);
    case "aov":
      return rub(Math.round(aov(row)));
    case "paidOrders":
      return num(row.paidOrders);
    case "inTransitOrders":
      return num(row.inTransitOrders);
    case "completedOrders":
      return num(row.completedOrders);
    case "canceledOrders":
      return num(row.canceledOrders);
    case "g2n":
      return pct(g2n(row));
    case "p2n":
      return pct(p2n(row));
  }
}

function cellClass(key: SortKey, row: CustomerRow): string {
  const base = "px-2 py-2.5 text-right tabular-nums";
  if (key === "revenue") return `${base} font-medium text-ink`;
  if (key === "inTransitOrders" && row.inTransitOrders > 0) return `${base} font-medium text-amber-600`;
  if (key === "canceledOrders" && row.canceledOrders > 0) return `${base} text-rose-500`;
  if (key === "g2n" || key === "p2n") return `${base} text-slate-600`;
  return `${base} text-slate-600`;
}

export default function CustomerTable({ rows, totalRevenue }: Props) {
  const [sortKey, setSortKey] = useState<SortKey>("revenue");
  const [sortDesc, setSortDesc] = useState(true);

  const sorted = useMemo(() => {
    const copy = [...rows];
    copy.sort((a, b) => {
      const diff = sortValue(b, sortKey) - sortValue(a, sortKey);
      return sortDesc ? diff : -diff;
    });
    return copy;
  }, [rows, sortKey, sortDesc]);

  const maxShare = Math.max(1, ...rows.map((r) => r.revenueShare));

  const onSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDesc((v) => !v);
    } else {
      setSortKey(key);
      setSortDesc(true);
    }
  };

  return (
    <div className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
      <div className="mb-4 flex flex-wrap items-baseline justify-between gap-2">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Топ покупателей</h2>
        <p className="text-xs text-slate-400">
          Воронка заказов: оформлено → оплата → транзит → выкуп · всего {rub(totalRevenue)}
        </p>
      </div>

      {rows.length === 0 ? (
        <div className="text-sm text-slate-400">Нет данных за период</div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[880px] text-sm">
            <thead>
              <tr className="border-b border-slate-100 text-[11px] uppercase tracking-wide text-slate-400">
                <th className="pb-2 pr-3 text-left font-medium">#</th>
                <th className="pb-2 pr-3 text-left font-medium">Покупатель</th>
                {COLUMNS.map((col) => (
                  <th key={col.key} className="px-2 pb-2 text-right font-medium">
                    <button
                      type="button"
                      onClick={() => onSort(col.key)}
                      title={col.title}
                      className={`ml-auto inline-flex items-center gap-0.5 hover:text-slate-600 ${
                        sortKey === col.key ? "text-brand" : ""
                      }`}
                    >
                      {col.label}
                      {sortKey === col.key && <span className="text-[10px]">{sortDesc ? "▼" : "▲"}</span>}
                    </button>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {sorted.map((row, i) => (
                <tr key={row.name} className="border-b border-slate-50 last:border-0">
                  <td className="py-2.5 pr-3 tabular-nums text-slate-400">{i + 1}</td>
                  <td className="max-w-[180px] py-2.5 pr-3">
                    <div className="flex items-center gap-1.5">
                      <span className="truncate text-slate-700" title={row.name}>
                        {row.name}
                      </span>
                      {row.orders > 1 && (
                        <span className="shrink-0 rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium text-slate-500">
                          ×{row.orders}
                        </span>
                      )}
                    </div>
                    <div className="mt-1 h-1 overflow-hidden rounded bg-slate-100">
                      <div
                        className="h-full bg-brand/70"
                        style={{ width: `${(row.revenueShare / maxShare) * 100}%` }}
                      />
                    </div>
                  </td>
                  {COLUMNS.map((col) => (
                    <td key={col.key} className={cellClass(col.key, row)}>
                      {cellValue(row, col.key)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
