import { useEffect, useMemo, useRef, useState, type CSSProperties } from "react";
import type { City } from "../types";
import { num } from "../utils/format";

interface Props {
  label: string;
  allLabel: string; // подпись для «ничего не выбрано», напр. «Все города»
  options: City[];
  selected: string[];
  onChange: (values: string[]) => void;
  width?: number;
}

export default function MultiSelect({
  label,
  allLabel,
  options,
  selected,
  onChange,
  width = 200,
}: Props) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDocClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDocClick);
    return () => document.removeEventListener("mousedown", onDocClick);
  }, [open]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return options;
    return options.filter((o) => o.name.toLowerCase().includes(q));
  }, [options, query]);

  const selectedSet = useMemo(() => new Set(selected), [selected]);

  const toggle = (name: string) => {
    if (selectedSet.has(name)) {
      onChange(selected.filter((s) => s !== name));
    } else {
      onChange([...selected, name]);
    }
  };

  const summary =
    selected.length === 0
      ? allLabel
      : selected.length === 1
      ? selected[0]
      : `Выбрано: ${selected.length}`;

  return (
    <div
      className="flex w-full flex-col text-xs font-medium text-slate-500 sm:w-auto"
      ref={ref}
      style={{
        "--filter-width": `${width}px`,
        "--filter-menu-width": `${Math.max(width, 220)}px`,
      } as CSSProperties}
    >
      {label}
      <div className="relative mt-1 w-full">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="flex w-full items-center justify-between gap-2 rounded-lg border border-slate-300 px-3 py-1.5 text-left text-sm text-ink transition hover:border-brand sm:w-[var(--filter-width)]"
        >
          <span className="truncate">{summary}</span>
          <span className="shrink-0 text-slate-400">▾</span>
        </button>
        {open && (
          <div
            className="absolute z-20 mt-1 w-full rounded-lg border border-slate-200 bg-white p-2 shadow-lg sm:w-[var(--filter-menu-width)]"
          >
            <input
              autoFocus
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Поиск…"
              className="mb-2 w-full rounded-md border border-slate-200 px-2 py-1 text-sm text-ink"
            />
            <div className="flex items-center justify-between px-1 pb-1.5">
              <span className="text-[11px] text-slate-400">
                {selected.length > 0 ? `${selected.length} выбрано` : allLabel}
              </span>
              {selected.length > 0 && (
                <button
                  type="button"
                  onClick={() => onChange([])}
                  className="text-[11px] font-medium text-brand hover:underline"
                >
                  Сбросить
                </button>
              )}
            </div>
            <div className="max-h-64 overflow-auto">
              {filtered.length === 0 && (
                <div className="px-2 py-3 text-center text-xs text-slate-400">Ничего не найдено</div>
              )}
              {filtered.map((o) => (
                <label
                  key={o.name}
                  className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm text-ink hover:bg-slate-50"
                >
                  <input
                    type="checkbox"
                    checked={selectedSet.has(o.name)}
                    onChange={() => toggle(o.name)}
                    className="h-3.5 w-3.5 rounded border-slate-300 text-brand focus:ring-brand"
                  />
                  <span className="flex-1 truncate">{o.name}</span>
                  <span className="shrink-0 text-xs text-slate-400">{num(o.orders)}</span>
                </label>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
