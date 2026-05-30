import type { FunnelReport } from "../types";
import { num, pct } from "../utils/format";

interface Props {
  report: FunnelReport;
}

export default function ProblemsPanel({ report }: Props) {
  const cancelPct = report.gross > 0 ? (report.canceled / report.gross) * 100 : 0;

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <div className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
        <h2 className="mb-4 text-sm font-semibold uppercase tracking-wide text-slate-500">
          Отмены и неоплата
        </h2>
        <div className="grid grid-cols-3 gap-3">
          <Stat label="Отменено" value={`${num(report.canceled)}`} sub={pct(cancelPct)} tone="bad" />
          <Stat label="Возвраты" value={num(report.returns)} tone="warn" />
          <Stat label="Проблемы сборки" value={num(report.problems)} tone="warn" />
        </div>
        <div className="mt-4">
          <div className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-400">
            Причины отмены
          </div>
          {report.canceledNoReason > 0 && (
            <div className="mb-2 rounded-lg bg-slate-50 px-3 py-2 text-sm text-slate-500">
              Причина не указана: <span className="font-medium text-slate-700">{num(report.canceledNoReason)}</span>{" "}
              заказов — в основном брошенная онлайн-оплата.
            </div>
          )}
          <ul className="space-y-1.5">
            {report.topCancelReasons.map((r) => (
              <li key={r.label} className="flex justify-between text-sm">
                <span className="truncate pr-2 text-slate-600" title={r.label}>{r.label}</span>
                <span className="shrink-0 font-medium text-ink">{num(r.orders)}</span>
              </li>
            ))}
            {report.topCancelReasons.length === 0 && (
              <li className="text-sm text-slate-400">Заполненных причин нет</li>
            )}
          </ul>
        </div>
      </div>

      <div className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
        <h2 className="mb-4 text-sm font-semibold uppercase tracking-wide text-slate-500">
          Проблемы на сборке/отгрузке
        </h2>
        <ul className="space-y-1.5">
          {report.topProblems.map((p) => (
            <li key={p.label} className="flex justify-between gap-2 text-sm">
              <span className="truncate text-slate-600" title={p.label}>{p.label}</span>
              <span className="shrink-0 font-medium text-ink">{num(p.orders)}</span>
            </li>
          ))}
          {report.topProblems.length === 0 && (
            <li className="text-sm text-slate-400">Проблемных заказов за период нет</li>
          )}
        </ul>
      </div>
    </div>
  );
}

function Stat({ label, value, sub, tone }: { label: string; value: string; sub?: string; tone: "bad" | "warn" }) {
  const color = tone === "bad" ? "text-rose-600" : "text-amber-600";
  return (
    <div className="rounded-lg bg-slate-50 p-3 text-center">
      <div className="text-xs text-slate-500">{label}</div>
      <div className={`mt-1 text-xl font-semibold ${color}`}>{value}</div>
      {sub && <div className="text-xs text-slate-400">{sub}</div>}
    </div>
  );
}
