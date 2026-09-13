import type { Bounds, GoalChannelSummary, GoalPeriodSummary } from "../types";

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

function localDate(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function parseDate(value: string): Date {
  return new Date(`${value}T00:00:00`);
}

function daysBetween(start: string, end: string): number {
  return Math.floor((parseDate(end).getTime() - parseDate(start).getTime()) / 86_400_000) + 1;
}

export function monthRange(year: number, month: number): { start: string; end: string } {
  const m = String(month).padStart(2, "0");
  const lastDay = new Date(year, month, 0).getDate();
  return { start: `${year}-${m}-01`, end: `${year}-${m}-${String(lastDay).padStart(2, "0")}` };
}

function periodChannel(period: GoalPeriodSummary | null, channel: PlanChannel): GoalChannelSummary | undefined {
  return period?.channels.find((item) => item.channel === channel);
}

export function buildAllocation(
  historical: GoalPeriodSummary | null,
  current: GoalPeriodSummary
): AllocationBasis {
  const historicalSite = periodChannel(historical, "site")?.revenue ?? 0;
  const historicalApp = periodChannel(historical, "app")?.revenue ?? 0;
  let site = historicalSite;
  let app = historicalApp;
  let source: AllocationBasis["source"] = "history";

  if (site + app <= 0) {
    site = periodChannel(current, "site")?.revenue ?? 0;
    app = periodChannel(current, "app")?.revenue ?? 0;
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
    start: historical?.start ?? "",
    end: historical?.end ?? "",
    source,
  };
}

function ratio(numerator: number, denominator: number): number {
  return denominator > 0 ? numerator / denominator : 0;
}

function selectDriver(
  current: GoalChannelSummary,
  historical: GoalChannelSummary | null
): GoalDriver {
  const currentG2N = ratio(current.netRevenue, current.revenue);
  const historicalG2N = historical
    ? ratio(historical.netRevenue, historical.revenue)
    : 0;

  if (current.sessions > 0 && current.aov > 0) {
    return {
      sessions: current.sessions,
      netCr: current.netCr / 100,
      aov: current.aov,
      g2n: historicalG2N || currentG2N,
      source: "current",
    };
  }
  if (historical && historical.sessions > 0 && historical.aov > 0) {
    return {
      sessions: historical.sessions,
      netCr: historical.netCr / 100,
      aov: historical.aov,
      g2n: historicalG2N,
      source: "history",
    };
  }
  return {
    sessions: current.sessions,
    netCr: 0,
    aov: current.aov || historical?.aov || 0,
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
  current: GoalPeriodSummary;
  historical: GoalPeriodSummary | null;
  today?: Date;
}): GoalProgress {
  const dates = analysisDates(args.year, args.month, args.bounds, args.today);
  const currentAll = periodChannel(args.current, "all")!;
  const historicalAll = periodChannel(args.historical, "all") ?? null;
  const driver = selectDriver(currentAll, historicalAll);
  const factNet = currentAll.netRevenue;
  const projectedNet = Math.max(factNet, Math.round(currentAll.revenue * driver.g2n));
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

  const currentSite = periodChannel(args.current, "site")!;
  const currentApp = periodChannel(args.current, "app")!;
  const factTotal = currentSite.netRevenue + currentApp.netRevenue;
  const channelInputs = [
    { key: "site" as const, label: "Сайт", share: args.allocation.siteShare, summary: currentSite },
    { key: "app" as const, label: "Приложение", share: args.allocation.appShare, summary: currentApp },
  ];
  const channels = channelInputs.map(({ key, label, share, summary }) => {
    const channelFact = summary.netRevenue;
    const target = key === "site"
      ? Math.round(args.target * share)
      : args.target - Math.round(args.target * args.allocation.siteShare);
    const projected = Math.max(channelFact, Math.round(summary.revenue * driver.g2n));
    return {
      channel: key,
      label,
      target,
      factNet: channelFact,
      projectedNet: projected,
      forecastNet: dates.elapsedDays > 0 ? Math.round(projected / dates.elapsedDays * dates.daysInMonth) : null,
      completionPct: target > 0 ? channelFact / target * 100 : 0,
      sessions: summary.sessions,
      netCr: summary.netCr,
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
