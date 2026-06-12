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

export function rubAbs(n: number): string {
  const sign = n > 0 ? "+" : "";
  const abs = Math.abs(n);
  if (abs >= 1_000_000) {
    const val = n / 1_000_000;
    return `${sign}${val.toLocaleString("ru-RU", { maximumFractionDigits: 1 })} млн ₽`;
  }
  if (abs >= 1_000) {
    const val = n / 1_000;
    return `${sign}${val.toLocaleString("ru-RU", { maximumFractionDigits: 1 })} тыс. ₽`;
  }
  return `${sign}${new Intl.NumberFormat("ru-RU").format(n)} ₽`;
}

export function numAbs(n: number): string {
  const sign = n > 0 ? "+" : "";
  const abs = Math.abs(n);
  if (abs >= 1_000_000) {
    const val = n / 1_000_000;
    return `${sign}${val.toLocaleString("ru-RU", { maximumFractionDigits: 1 })} млн`;
  }
  if (abs >= 10_000) {
    const val = n / 1_000;
    return `${sign}${val.toLocaleString("ru-RU", { maximumFractionDigits: 1 })} тыс.`;
  }
  return `${sign}${new Intl.NumberFormat("ru-RU").format(n)}`;
}

export function ppAbs(n: number): string {
  const sign = n > 0 ? "+" : "";
  return `${sign}${n.toLocaleString("ru-RU", { maximumFractionDigits: 1 })} п.п.`;
}

export function floatAbs(n: number): string {
  const sign = n > 0 ? "+" : "";
  return `${sign}${n.toLocaleString("ru-RU", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

