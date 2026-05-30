import type { Delta } from "../utils/format";
import { deltaText } from "../utils/format";

interface Props {
  d: Delta;
  invert?: boolean; // true: рост = плохо (например, отмены)
}

export default function DeltaBadge({ d, invert }: Props) {
  const good = invert ? d.dir === "down" : d.dir === "up";
  const color =
    d.dir === "flat"
      ? "text-slate-400 bg-slate-100"
      : good
      ? "text-emerald-700 bg-emerald-50"
      : "text-rose-700 bg-rose-50";
  const arrow = d.dir === "up" ? "▲" : d.dir === "down" ? "▼" : "■";
  return (
    <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${color}`}>
      <span>{arrow}</span>
      {deltaText(d)}
    </span>
  );
}
