import type { FunnelReport, SegmentRow } from "../types";
import { num, pct } from "../utils/format";
import FunnelChart from "./FunnelChart";
import SegmentFunnel from "./SegmentFunnel";
import ProblemsPanel from "./ProblemsPanel";

interface Props {
  report: FunnelReport;
}

// worstPayment ищет способ оплаты с худшей оплачиваемостью среди значимых по объёму.
function worstPayment(report: FunnelReport): SegmentRow | null {
  const payment = report.segments.find((s) => s.by === "payment");
  if (!payment) return null;
  const significant = payment.rows.filter((r) => r.gross >= 20);
  if (significant.length === 0) return null;
  return significant.reduce((w, r) => (r.cancelRate > w.cancelRate ? r : w));
}

export default function FunnelTab({ report }: Props) {
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

      <FunnelChart stages={report.stages} />
      <SegmentFunnel segments={report.segments} />
      <ProblemsPanel report={report} />
    </div>
  );
}
