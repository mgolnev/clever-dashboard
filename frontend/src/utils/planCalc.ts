import type { AcquisitionChannel, AcquisitionReport, Bounds, KPI, NamedCount, Report } from "../types";

export type PlanChannel = "all" | "site" | "app";

export interface AllocationBasis {
  siteShare: number;
  appShare: number;
  start: string;
  end: string;
  source: "history" | "current" | "default";
}

export interface GoalDriver {
  sessions: number;
  netCr: number;
  aov: number;
  g2n: number;
  source: "current" | "history" | "missing";
}

export interface ChannelProgress {
  channel: "site" | "app";
  label: string;
  target: number;
  factNet: number;
  projectedNet: number;
  forecastNet: number | null;
  completionPct: number;
  sessions: number;
  netCr: number;
  factShare: number;
}

export interface GoalProgress {
  target: number;
  factNet: number;
  projectedNet: number;
  planToDate: number;
  forecastNet: number | null;
  completionPct: number;
  pacePct: number;
  paceDelta: number;
  remainingRevenue: number;
  requiredRevenuePerDay: number | null;
  requiredOrdersPerDay: number | null;
  requiredSessionsPerDay: number | null;
  elapsedDays: number;
  remainingDays: number;
  daysInMonth: number;
  dataThrough: string | null;
  coverageStarts: string | null;
  driver: GoalDriver;
  channels: ChannelProgress[];
}

export const emptyPlanFilters = {
  city: [] as string[],
  region: [] as string[],
  channel: [] as string[],
  payment: [] as string[],
  delivery: [] as string[],
  coupon: [] as string[],
};

function localDate(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function parseDate(value: string): Date {
  return new Date(`${value}T00:00:00`);
}

function addDays(value: string, days: number): string {
  const date = parseDate(value);
  date.setDate(date.getDate() + days);
  return localDate(date);
}

function daysBetween(start: string, end: string): number {
  return Math.floor((parseDate(end).getTime() - parseDate(start).getTime()) / 86_400_000) + 1;
}

export function monthRange(year: number, month: number): { start: string; end: string } {
  const m = String(month).padStart(2, "0");
  const lastDay = new Date(year, month, 0).getDate();
  return { start: `${year}-${m}-01`, end: `${year}-${m}-${String(lastDay).padStart(2, "0")}` };
}

export function allocationRange(year: number, month: number, bounds: Bounds): { start: string; end: string } | null {
  if (!bounds.max) return null;
  const { start: monthStart } = monthRange(year, month);
  const end = bounds.max < addDays(monthStart, -1) ? bounds.max : addDays(monthStart, -1);
  if (bounds.min && end < bounds.min) return null;
  const candidateStart = addDays(end, -29);
  return { start: bounds.min && candidateStart < bounds.min ? bounds.min : candidateStart, end };
}

function channelRevenue(rows: NamedCount[], channel: "Сайт" | "Приложение"): number {
  return rows.find((row) => row.name === channel)?.revenue ?? 0;
}

export function buildAllocation(
  historical: Report | null,
  current: Report,
  range: { start: string; end: string } | null
): AllocationBasis {
  const historicalSite = historical ? channelRevenue(historical.current.byChannel, "Сайт") : 0;
  const historicalApp = historical ? channelRevenue(historical.current.byChannel, "Приложение") : 0;
  let site = historicalSite;
  let app = historicalApp;
  let source: AllocationBasis["source"] = "history";

  if (site + app <= 0) {
    site = channelRevenue(current.current.byChannel, "Сайт");
    app = channelRevenue(current.current.byChannel, "Приложение");
    source = "current";
  }
  if (site + app <= 0) {
    site = 1;
    app = 1;
    source = "default";
  }

  const total = site + app;
  return {
    siteShare: site / total,
    appShare: app / total,
    start: range?.start ?? "",
    end: range?.end ?? "",
    source,
  };
}

function ratio(numerator: number, denominator: number): number {
  return denominator > 0 ? numerator / denominator : 0;
}

function channel(report: AcquisitionReport | null, key: PlanChannel): AcquisitionChannel | undefined {
  return report?.current.channels.find((item) => item.channel === key);
}

function selectDriver(
  currentMetrics: KPI,
  currentAcquisition: AcquisitionReport,
  historicalMetrics: KPI | null,
  historicalAcquisition: AcquisitionReport | null
): GoalDriver {
  const currentTraffic = channel(currentAcquisition, "all");
  const historicalTraffic = channel(historicalAcquisition, "all");
  const currentG2N = ratio(currentMetrics.stages.redeemedNet.revenue, currentMetrics.revenue);
  const historicalG2N = historicalMetrics
    ? ratio(historicalMetrics.stages.redeemedNet.revenue, historicalMetrics.revenue)
    : 0;

  if (currentTraffic && currentTraffic.sessions > 0 && currentMetrics.aov > 0) {
    return {
      sessions: currentTraffic.sessions,
      netCr: currentTraffic.netCr / 100,
      aov: currentMetrics.aov,
      g2n: historicalG2N || currentG2N,
      source: "current",
    };
  }
  if (historicalTraffic && historicalTraffic.sessions > 0 && historicalMetrics?.aov) {
    return {
      sessions: historicalTraffic.sessions,
      netCr: historicalTraffic.netCr / 100,
      aov: historicalMetrics.aov,
      g2n: historicalG2N,
      source: "history",
    };
  }
  return {
    sessions: currentTraffic?.sessions ?? 0,
    netCr: 0,
    aov: currentMetrics.aov || historicalMetrics?.aov || 0,
    g2n: historicalG2N || currentG2N,
    source: "missing",
  };
}

function analysisDates(year: number, month: number, bounds: Bounds, today = new Date()) {
  const range = monthRange(year, month);
  const todayValue = localDate(today);
  const naturalEnd = range.end < todayValue ? range.end : todayValue;
  const dataThrough = bounds.max && bounds.max >= range.start
    ? (bounds.max < naturalEnd ? bounds.max : naturalEnd)
    : null;
  const coverageStarts = bounds.min && bounds.min <= range.end
    ? (bounds.min > range.start ? bounds.min : range.start)
    : null;
  const elapsedDays = dataThrough ? daysBetween(range.start, dataThrough) : 0;
  const daysInMonth = daysBetween(range.start, range.end);
  return {
    dataThrough,
    coverageStarts,
    elapsedDays,
    daysInMonth,
    remainingDays: Math.max(0, daysInMonth - elapsedDays),
  };
}

export function buildGoalProgress(args: {
  year: number;
  month: number;
  target: number;
  bounds: Bounds;
  allocation: AllocationBasis;
  metricsAll: Report;
  metricsSite: Report;
  metricsApp: Report;
  acquisition: AcquisitionReport;
  historicalMetrics: Report | null;
  historicalAcquisition: AcquisitionReport | null;
  today?: Date;
}): GoalProgress {
  const dates = analysisDates(args.year, args.month, args.bounds, args.today);
  const kpi = args.metricsAll.current.kpi;
  const historicalKpi = args.historicalMetrics?.current.kpi ?? null;
  const driver = selectDriver(kpi, args.acquisition, historicalKpi, args.historicalAcquisition);
  const factNet = kpi.stages.redeemedNet.revenue;
  const projectedNet = Math.max(factNet, Math.round(kpi.revenue * driver.g2n));
  const planToDate = dates.daysInMonth > 0
    ? Math.round(args.target * dates.elapsedDays / dates.daysInMonth)
    : 0;
  const forecastNet = dates.elapsedDays > 0
    ? Math.round(projectedNet / dates.elapsedDays * dates.daysInMonth)
    : null;
  const remainingRevenue = Math.max(0, args.target - projectedNet);
  const revenuePerOrder = driver.aov * driver.g2n;
  const revenuePerSession = driver.netCr * revenuePerOrder;
  const requiredRevenuePerDay = dates.remainingDays > 0 && args.target > 0
    ? Math.ceil(remainingRevenue / dates.remainingDays)
    : null;
  const requiredOrdersPerDay = requiredRevenuePerDay != null && revenuePerOrder > 0
    ? Math.ceil(requiredRevenuePerDay / revenuePerOrder)
    : null;
  const requiredSessionsPerDay = requiredRevenuePerDay != null && revenuePerSession > 0
    ? Math.ceil(requiredRevenuePerDay / revenuePerSession)
    : null;

  const factTotal = args.metricsSite.current.kpi.stages.redeemedNet.revenue
    + args.metricsApp.current.kpi.stages.redeemedNet.revenue;
  const channelInputs = [
    { key: "site" as const, label: "Сайт", share: args.allocation.siteShare, report: args.metricsSite },
    { key: "app" as const, label: "Приложение", share: args.allocation.appShare, report: args.metricsApp },
  ];
  const channels = channelInputs.map(({ key, label, share, report }) => {
    const channelKpi = report.current.kpi;
    const channelFact = channelKpi.stages.redeemedNet.revenue;
    const target = key === "site"
      ? Math.round(args.target * share)
      : args.target - Math.round(args.target * args.allocation.siteShare);
    const projected = Math.max(channelFact, Math.round(channelKpi.revenue * driver.g2n));
    const acquisitionChannel = channel(args.acquisition, key);
    return {
      channel: key,
      label,
      target,
      factNet: channelFact,
      projectedNet: projected,
      forecastNet: dates.elapsedDays > 0 ? Math.round(projected / dates.elapsedDays * dates.daysInMonth) : null,
      completionPct: target > 0 ? channelFact / target * 100 : 0,
      sessions: acquisitionChannel?.sessions ?? 0,
      netCr: acquisitionChannel?.netCr ?? 0,
      factShare: factTotal > 0 ? channelFact / factTotal : 0,
    };
  });

  return {
    target: args.target,
    factNet,
    projectedNet,
    planToDate,
    forecastNet,
    completionPct: args.target > 0 ? factNet / args.target * 100 : 0,
    pacePct: planToDate > 0 ? factNet / planToDate * 100 : 0,
    paceDelta: factNet - planToDate,
    remainingRevenue,
    requiredRevenuePerDay,
    requiredOrdersPerDay,
    requiredSessionsPerDay,
    elapsedDays: dates.elapsedDays,
    remainingDays: dates.remainingDays,
    daysInMonth: dates.daysInMonth,
    dataThrough: dates.dataThrough,
    coverageStarts: dates.coverageStarts,
    driver,
    channels,
  };
}

export function allocationTargets(total: number, allocation: AllocationBasis) {
  const site = Math.round(total * allocation.siteShare);
  return { all: total, site, app: total - site };
}

export function forecastNet(sessions: number, cr: number, aov: number, g2n: number): number {
  return sessions * cr * aov * g2n;
}
