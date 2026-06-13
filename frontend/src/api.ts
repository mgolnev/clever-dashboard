import type {
  Bounds,
  City,
  FunnelReport,
  ImportResult,
  LogisticsDynamics,
  LogisticsReport,
  PlanReport,
  Report,
  TrafficReport,
} from "./types";

async function handle<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let msg = `Ошибка ${res.status}`;
    try {
      const body = await res.json();
      if (body?.error) msg = body.error;
    } catch {
      /* ignore */
    }
    throw new Error(msg);
  }
  return res.json() as Promise<T>;
}

// Filters — сквозные фильтры дашборда (мультивыбор по каждому полю).
export interface Filters {
  city: string[];
  region: string[];
  channel: string[];
  payment: string[];
  delivery: string[];
  coupon: string[];
}

function query(
  start: string,
  end: string,
  f: Filters,
  opts?: { granularity?: string; compareStart?: string; compareEnd?: string }
): string {
  const p = new URLSearchParams({ start, end });
  if (opts?.granularity) p.set("granularity", opts.granularity);
  if (opts?.compareStart && opts?.compareEnd) {
    p.set("compareStart", opts.compareStart);
    p.set("compareEnd", opts.compareEnd);
  }
  if (f.city.length) p.set("city", f.city.join(","));
  if (f.region.length) p.set("region", f.region.join(","));
  if (f.channel.length) p.set("channel", f.channel.join(","));
  if (f.payment.length) p.set("payment", f.payment.join(","));
  if (f.delivery.length) p.set("delivery", f.delivery.join(","));
  if (f.coupon.length) p.set("coupon", f.coupon.join(","));
  return p.toString();
}

export const api = {
  bounds: () => fetch("/api/bounds").then((r) => handle<Bounds>(r)),

  cities: () => fetch("/api/cities").then((r) => handle<City[]>(r)),

  regions: () => fetch("/api/regions").then((r) => handle<City[]>(r)),

  channels: () => fetch("/api/channels").then((r) => handle<City[]>(r)),

  payments: () => fetch("/api/payments").then((r) => handle<City[]>(r)),

  deliveries: () => fetch("/api/deliveries").then((r) => handle<City[]>(r)),

  coupons: () => fetch("/api/coupons").then((r) => handle<City[]>(r)),

  metrics: (start: string, end: string, f: Filters, compareStart?: string, compareEnd?: string) =>
    fetch(`/api/metrics?${query(start, end, f, { compareStart, compareEnd })}`).then((r) =>
      handle<Report>(r)
    ),

  funnel: (start: string, end: string, f: Filters, compareStart?: string, compareEnd?: string) =>
    fetch(`/api/funnel?${query(start, end, f, { compareStart, compareEnd })}`).then((r) =>
      handle<FunnelReport>(r)
    ),

  logistics: (
    start: string,
    end: string,
    f: Filters,
    granularity?: string,
    compareStart?: string,
    compareEnd?: string
  ) =>
    fetch(
      `/api/logistics?${query(start, end, f, { granularity, compareStart, compareEnd })}`
    ).then((r) => handle<LogisticsReport>(r)),

  dynamics: (start: string, end: string, f: Filters, groupBy: string, granularity?: string) =>
    fetch(
      `/api/dynamics?${query(start, end, f, { granularity })}&groupBy=${encodeURIComponent(groupBy)}`
    ).then((r) => handle<LogisticsDynamics>(r)),

  importFile: (file: File) => {
    const fd = new FormData();
    fd.append("file", file);
    return fetch("/api/import", { method: "POST", body: fd }).then((r) =>
      handle<ImportResult>(r)
    );
  },

  getPlan: (year: number) =>
    fetch(`/api/plan?year=${year}`).then((r) => handle<PlanReport>(r)),

  putPlan: (
    year: number,
    items: { month: number; channel: string; netTarget: number }[]
  ) =>
    fetch("/api/plan", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ year, items }),
    }).then((r) => handle<PlanReport>(r)),

  getTraffic: (year: number) =>
    fetch(`/api/traffic?year=${year}`).then((r) => handle<TrafficReport>(r)),

  putTraffic: (
    year: number,
    items: { month: number; channel: string; visits: number }[]
  ) =>
    fetch("/api/traffic", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ year, items }),
    }).then((r) => handle<TrafficReport>(r)),

  localFiles: () => fetch("/api/import/local").then((r) => handle<string[]>(r)),

  importLocalFile: (filename: string) =>
    fetch("/api/import/local", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ filename }),
    }).then((r) => handle<ImportResult>(r)),
};
