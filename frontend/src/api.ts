import type { Bounds, City, FunnelReport, ImportResult, Report } from "./types";

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

export const api = {
  bounds: () => fetch("/api/bounds").then((r) => handle<Bounds>(r)),

  cities: () => fetch("/api/cities").then((r) => handle<City[]>(r)),

  metrics: (start: string, end: string, city: string) =>
    fetch(`/api/metrics?start=${start}&end=${end}&city=${encodeURIComponent(city)}`).then((r) =>
      handle<Report>(r)
    ),

  funnel: (start: string, end: string, city: string) =>
    fetch(`/api/funnel?start=${start}&end=${end}&city=${encodeURIComponent(city)}`).then((r) =>
      handle<FunnelReport>(r)
    ),

  importFile: (file: File) => {
    const fd = new FormData();
    fd.append("file", file);
    return fetch("/api/import", { method: "POST", body: fd }).then((r) =>
      handle<ImportResult>(r)
    );
  },
};
