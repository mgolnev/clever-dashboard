import { apiFetch } from "./services/auth";
import type {
  Bounds,
  AcquisitionReport,
  AnalyticsStatus,
  AnalyticsSyncTrigger,
  City,
  CustomerAnalyticsReport,
  CustomerGranularity,
  FunnelReport,
  GoalReport,
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

export type ImportProgress =
  | { phase: "uploading"; percent: number }
  | { phase: "processing"; percent: 100 };

interface ImportUploadSession {
  uploadId: string;
  filename: string;
  size: number;
  chunkSize: number;
  chunksTotal: number;
}

const directImportLimit = 4 * 1024 * 1024;

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
  goal: (year: number, month: number, signal?: AbortSignal) => {
    const params = new URLSearchParams({ year: String(year), month: String(month) });
    return apiFetch(`/api/goal?${params.toString()}`, { signal }).then((r) => handle<GoalReport>(r));
  },

  bounds: () => apiFetch("/api/bounds").then((r) => handle<Bounds>(r)),

  cities: () => apiFetch("/api/cities").then((r) => handle<City[]>(r)),

  regions: () => apiFetch("/api/regions").then((r) => handle<City[]>(r)),

  channels: () => apiFetch("/api/channels").then((r) => handle<City[]>(r)),

  payments: () => apiFetch("/api/payments").then((r) => handle<City[]>(r)),

  deliveries: () => apiFetch("/api/deliveries").then((r) => handle<City[]>(r)),

  coupons: () => apiFetch("/api/coupons").then((r) => handle<City[]>(r)),

  acquisition: (start: string, end: string, compareStart?: string, compareEnd?: string) => {
    const p = new URLSearchParams({ start, end });
    if (compareStart && compareEnd) {
      p.set("compareStart", compareStart);
      p.set("compareEnd", compareEnd);
    }
    return apiFetch(`/api/acquisition?${p.toString()}`).then((r) => handle<AcquisitionReport>(r));
  },

  analyticsStatus: (signal?: AbortSignal) =>
    apiFetch("/api/analytics/status", { signal }).then((r) => handle<AnalyticsStatus>(r)),

  syncAnalytics: (signal?: AbortSignal) =>
    apiFetch("/api/analytics/sync", { method: "POST", signal }).then((r) =>
      handle<AnalyticsSyncTrigger>(r)
    ),

  metrics: (start: string, end: string, f: Filters, compareStart?: string, compareEnd?: string) =>
    apiFetch(`/api/metrics?${query(start, end, f, { compareStart, compareEnd })}`).then((r) =>
      handle<Report>(r)
    ),

  customerAnalytics: (start: string, end: string, f: Filters, granularity: CustomerGranularity) =>
    apiFetch(`/api/customer-analytics?${query(start, end, f, { granularity })}`).then((r) =>
      handle<CustomerAnalyticsReport>(r)
    ),

  funnel: (start: string, end: string, f: Filters, compareStart?: string, compareEnd?: string) =>
    apiFetch(`/api/funnel?${query(start, end, f, { compareStart, compareEnd })}`).then((r) =>
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
    apiFetch(
      `/api/logistics?${query(start, end, f, { granularity, compareStart, compareEnd })}`
    ).then((r) => handle<LogisticsReport>(r)),

  dynamics: (start: string, end: string, f: Filters, groupBy: string, granularity?: string) =>
    apiFetch(
      `/api/dynamics?${query(start, end, f, { granularity })}&groupBy=${encodeURIComponent(groupBy)}`
    ).then((r) => handle<LogisticsDynamics>(r)),

  importFile: async (file: File, onProgress?: (progress: ImportProgress) => void) => {
    if (file.size <= directImportLimit) {
      const fd = new FormData();
      fd.append("file", file);
      onProgress?.({ phase: "processing", percent: 100 });
      return apiFetch("/api/import", { method: "POST", body: fd }).then((r) =>
        handle<ImportResult>(r)
      );
    }

    onProgress?.({ phase: "uploading", percent: 0 });
    const session = await apiFetch("/api/import/uploads", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ filename: file.name, size: file.size }),
    }).then((r) => handle<ImportUploadSession>(r));

    try {
      for (let index = 0; index < session.chunksTotal; index += 1) {
        const start = index * session.chunkSize;
        const end = Math.min(start + session.chunkSize, file.size);
        const response = await apiFetch(`/api/import/uploads/${session.uploadId}/chunks/${index}`, {
          method: "PUT",
          headers: { "Content-Type": "application/octet-stream" },
          body: file.slice(start, end),
        });
        if (!response.ok) await handle<never>(response);
        onProgress?.({
          phase: "uploading",
          percent: Math.round((end / file.size) * 100),
        });
      }
      onProgress?.({ phase: "processing", percent: 100 });
      return await apiFetch(`/api/import/uploads/${session.uploadId}/complete`, {
        method: "POST",
      }).then((r) => handle<ImportResult>(r));
    } catch (error) {
      await apiFetch(`/api/import/uploads/${session.uploadId}`, { method: "DELETE" }).catch(() => undefined);
      throw error;
    }
  },

  getPlan: (year: number) =>
    apiFetch(`/api/plan?year=${year}`).then((r) => handle<PlanReport>(r)),

  putPlan: (
    year: number,
    items: { month: number; channel: string; netTarget: number }[]
  ) =>
    apiFetch("/api/plan", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ year, items }),
    }).then((r) => handle<PlanReport>(r)),

  getTraffic: (year: number) =>
    apiFetch(`/api/traffic?year=${year}`).then((r) => handle<TrafficReport>(r)),

  putTraffic: (
    year: number,
    items: { month: number; channel: string; visits: number }[]
  ) =>
    apiFetch("/api/traffic", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ year, items }),
    }).then((r) => handle<TrafficReport>(r)),

  localFiles: () => apiFetch("/api/import/local").then((r) => handle<string[]>(r)),

  importLocalFile: (filename: string) =>
    apiFetch("/api/import/local", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ filename }),
    }).then((r) => handle<ImportResult>(r)),
};
