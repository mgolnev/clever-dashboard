import type { KPI, PlanMonth, TrafficMonth } from "../types";

export type PlanChannel = "all" | "site" | "app";

export interface ChannelMetrics {
  visits: number;
  orders: number;
  revenue: number;
  netRevenue: number;
  cr: number;
  aov: number;
  r: number;
}

export interface ChannelGoal {
  channel: PlanChannel;
  label: string;
  planTarget: number;
  factNet: number;
  gap: number;
  metrics: ChannelMetrics;
  requiredVisits: number | null;
  requiredVisitsPerDay: number | null;
  catchUpVisitsPerDay: number | null;
  factVisits: number;
}

const CHANNEL_LABELS: Record<PlanChannel, string> = {
  all: "Итого",
  site: "Сайт",
  app: "Приложение",
};

export function monthRange(year: number, month: number): { start: string; end: string } {
  const m = String(month).padStart(2, "0");
  const lastDay = new Date(year, month, 0).getDate();
  const d = String(lastDay).padStart(2, "0");
  return { start: `${year}-${m}-01`, end: `${year}-${m}-${d}` };
}

export function extractMetrics(kpi: KPI, visits: number): ChannelMetrics {
  // База воронки — не отменённые заказы: так CR/AOV согласованы с выручкой
  // (kpi.revenue считается без отмен) и AOV совпадает с kpi.aov бэкенда.
  const orders = kpi.netOrders;
  const revenue = kpi.revenue;
  const netRevenue = kpi.stages.completed.revenue;
  const cr = visits > 0 ? orders / visits : 0;
  const aov = orders > 0 ? revenue / orders : 0;
  const r = revenue > 0 ? netRevenue / revenue : 0;
  return { visits, orders, revenue, netRevenue, cr, aov, r };
}

export function requiredVisits(planTarget: number, m: ChannelMetrics): number | null {
  const denom = m.cr * m.aov * m.r;
  if (denom <= 0 || planTarget <= 0) return null;
  return Math.ceil(planTarget / denom);
}

/** Оставшиеся дни месяца для расчёта «догнать» (включая сегодня). */
export function remainingDays(year: number, month: number, today = new Date()): number {
  const daysInMonth = new Date(year, month, 0).getDate();
  const ty = today.getFullYear();
  const tm = today.getMonth() + 1;
  if (year < ty || (year === ty && month < tm)) return daysInMonth;
  if (year > ty || (year === ty && month > tm)) return daysInMonth;
  const dayOfMonth = today.getDate();
  return Math.max(1, daysInMonth - dayOfMonth + 1);
}

export function buildChannelGoals(
  planMonth: PlanMonth,
  trafficMonth: TrafficMonth,
  metricsAll: KPI,
  metricsSite: KPI,
  metricsApp: KPI,
  year: number,
  month: number
): ChannelGoal[] {
  const siteVisits = trafficMonth.site;
  const appVisits = trafficMonth.app;
  const allVisits = siteVisits + appVisits;

  const defs: { ch: PlanChannel; target: number; visits: number; kpi: KPI }[] = [
    { ch: "all", target: planMonth.targets.all, visits: allVisits, kpi: metricsAll },
    { ch: "site", target: planMonth.targets.site, visits: siteVisits, kpi: metricsSite },
    { ch: "app", target: planMonth.targets.app, visits: appVisits, kpi: metricsApp },
  ];

  const remDays = remainingDays(year, month);
  const perDayBase = planMonth.daysInMonth;

  return defs.map(({ ch, target, visits, kpi }) => {
    const m = extractMetrics(kpi, visits);
    const req = requiredVisits(target, m);
    const reqPerDay = req != null ? Math.ceil(req / perDayBase) : null;
    const gap = target - m.netRevenue;
    let catchUp: number | null = null;
    if (req != null && visits < req) {
      catchUp = Math.ceil((req - visits) / remDays);
    }
    return {
      channel: ch,
      label: CHANNEL_LABELS[ch],
      planTarget: target,
      factNet: m.netRevenue,
      gap,
      metrics: m,
      requiredVisits: req,
      requiredVisitsPerDay: reqPerDay,
      catchUpVisitsPerDay: catchUp,
      factVisits: visits,
    };
  });
}

export function forecastNet(m: ChannelMetrics): number {
  return m.visits * m.cr * m.aov * m.r;
}
