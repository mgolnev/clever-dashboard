import type { FunnelReport, SegmentRow } from "../types";
import { num, pct, rub } from "../utils/format";
import FunnelChart from "./FunnelChart";
import SegmentFunnel from "./SegmentFunnel";
import ProblemsPanel from "./ProblemsPanel";

interface Props {
  report: FunnelReport;
  showCompare?: boolean;
}

// worstPayment ищет способ оплаты с худшей оплачиваемостью среди значимых по объёму.
function worstPayment(report: FunnelReport): SegmentRow | null {
  const payment = report.segments.find((s) => s.by === "payment");
  if (!payment) return null;
  const significant = payment.rows.filter((r) => r.gross >= 20);
  if (significant.length === 0) return null;
  return significant.reduce((w, r) => (r.cancelRate > w.cancelRate ? r : w));
}

export default function FunnelTab({ report, showCompare = true }: Props) {
  const paid = report.stages.find((s) => s.key === "paid");
  const worst = worstPayment(report);

  return (
    <div className="space-y-4">
      <div className="rounded-xl border border-indigo-100 bg-indigo-50/60 p-4">
        <div className="text-sm font-semibold text-indigo-900">Что показывает воронка</div>
        <ul className="mt-1 space-y-1 text-sm text-indigo-900/80">
          {paid && (
            <li>
              Главный отвал — на оплате: из {num(report.gross)} гросс-заказов оплачено{" "}
              <b>{pct(paid.fromStart)}</b>, теряем <b>{num(report.gross - paid.orders)}</b> заказов
              (отмены/неоплата), причём у {num(report.canceledNoReason)} причина не указана.
            </li>
          )}
          {worst && (
            <li>
              Самый проблемный способ оплаты — <b>{worst.name}</b>: отмена{" "}
              <b className="text-rose-700">{pct(worst.cancelRate)}</b> при оплате {pct(worst.paidRate)}.
              Стоит разобрать сценарий оплаты для него.
            </li>
          )}
        </ul>
      </div>

      <FunnelChart stages={report.stages} prevStages={report.prevStages} showCompare={showCompare} />
      <div className="rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">Итог выкупа</h2>
        <div className="mt-3 grid gap-3 sm:grid-cols-3">
          <div>
            <div className="text-xs text-slate-500">Выкуплено валово</div>
            <div className="mt-1 text-lg font-semibold text-ink">{num(report.redeemedGross)} зак.</div>
          </div>
          <div>
            <div className="text-xs text-rose-600">Возвраты</div>
            <div className="mt-1 text-lg font-semibold text-rose-700">{num(report.returns)} зак. · {rub(report.refundAmount)}</div>
          </div>
          <div>
            <div className="text-xs text-emerald-700">Выкуплено чистыми</div>
            <div className="mt-1 text-lg font-semibold text-emerald-700">{num(report.redeemedNet)} зак. · {rub(report.redeemedNetRevenue)}</div>
          </div>
        </div>
        <p className="mt-3 text-xs text-slate-400">Частичный возврат уменьшает только сумму; из количества исключаются лишь полностью возвращённые заказы.</p>
      </div>
      <SegmentFunnel segments={report.segments} />
      <ProblemsPanel report={report} />
    </div>
  );
}
