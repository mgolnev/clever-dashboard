import type { AcquisitionDay } from "../types";

export type AcquisitionGranularity = "day" | "week" | "month";

function bucketStart(day: string, granularity: AcquisitionGranularity): string {
  if (granularity === "day") return day;
  if (granularity === "month") return `${day.slice(0, 7)}-01`;

  const date = new Date(`${day}T00:00:00Z`);
  const daysSinceMonday = (date.getUTCDay() + 6) % 7;
  date.setUTCDate(date.getUTCDate() - daysSinceMonday);
  return date.toISOString().slice(0, 10);
}

export function aggregateAcquisitionDays(
  points: AcquisitionDay[],
  granularity: AcquisitionGranularity
): AcquisitionDay[] {
  if (granularity === "day") return points;

  const buckets = new Map<string, AcquisitionDay>();
  for (const point of points) {
    const day = bucketStart(point.day, granularity);
    const bucket = buckets.get(day) ?? {
      day,
      siteSessions: 0,
      appSessions: 0,
      siteUsers: 0,
      appUsers: 0,
      siteOrders: 0,
      appOrders: 0,
      sitePaidOrders: 0,
      appPaidOrders: 0,
    };
    bucket.siteSessions += point.siteSessions;
    bucket.appSessions += point.appSessions;
    bucket.siteUsers += point.siteUsers;
    bucket.appUsers += point.appUsers;
    bucket.siteOrders += point.siteOrders;
    bucket.appOrders += point.appOrders;
    bucket.sitePaidOrders += point.sitePaidOrders;
    bucket.appPaidOrders += point.appPaidOrders;
    buckets.set(day, bucket);
  }
  return Array.from(buckets.values()).sort((a, b) => a.day.localeCompare(b.day));
}

export function acquisitionBucketLabel(
  day: string,
  granularity: AcquisitionGranularity
): string {
  const date = new Date(`${day}T00:00:00`);
  if (granularity === "month") {
    return date.toLocaleDateString("ru-RU", { month: "short", year: "numeric" });
  }
  return date.toLocaleDateString("ru-RU", { day: "2-digit", month: "short" });
}
