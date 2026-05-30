export function rub(n: number): string {
  return new Intl.NumberFormat("ru-RU").format(n) + " ₽";
}

export function num(n: number): string {
  return new Intl.NumberFormat("ru-RU").format(n);
}

export function pct(n: number): string {
  return `${n.toLocaleString("ru-RU", { maximumFractionDigits: 1 })}%`;
}

export interface Delta {
  diff: number;
  ratio: number | null;
  dir: "up" | "down" | "flat";
}

// Сравнение текущего значения с предыдущим периодом.
export function delta(current: number, previous: number): Delta {
  const diff = current - previous;
  const ratio = previous === 0 ? null : (diff / previous) * 100;
  const dir = diff > 0 ? "up" : diff < 0 ? "down" : "flat";
  return { diff, ratio, dir };
}

export function deltaText(d: Delta): string {
  if (d.ratio === null) return d.diff === 0 ? "—" : "новое";
  const sign = d.ratio > 0 ? "+" : "";
  return `${sign}${d.ratio.toLocaleString("ru-RU", { maximumFractionDigits: 1 })}%`;
}
