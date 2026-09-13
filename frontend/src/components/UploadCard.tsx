import { useRef, useState } from "react";
import { api, type ImportProgress } from "../api";
import type { ImportResult } from "../types";
import { num } from "../utils/format";

interface Props {
  onImported: () => void;
}

export default function UploadCard({ onImported }: Props) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [progress, setProgress] = useState<ImportProgress | null>(null);

  async function upload(file: File) {
    setBusy(true);
    setError(null);
    setResult(null);
    try {
      const res = await api.importFile(file, setProgress);
      setResult(res);
      onImported();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка загрузки");
    } finally {
      setBusy(false);
      setProgress(null);
      if (inputRef.current) inputRef.current.value = "";
    }
  }

  const label = busy
    ? progress?.phase === "uploading"
      ? `Загрузка ${progress.percent}%`
      : "Обработка…"
    : error
      ? "Ошибка загрузки — повторить"
      : result
        ? `Добавлено ${num(result.ordersAdded)} · обновлено ${num(result.ordersUpdated)}${
            result.ordersSkipped > 0 ? ` · пропущено ${num(result.ordersSkipped)}` : ""
          }`
        : "Загрузить данные";

  return (
    <>
      <input
        ref={inputRef}
        type="file"
        accept=".xls,.csv,.html,.htm"
        disabled={busy}
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) upload(file);
        }}
        className="sr-only"
      />
      <button
        type="button"
        disabled={busy}
        onClick={() => inputRef.current?.click()}
        title={error || undefined}
        className={`inline-flex min-h-9 items-center gap-2 rounded-lg border px-3 py-1.5 text-sm font-semibold transition focus:outline-none focus:ring-2 disabled:cursor-wait disabled:opacity-70 ${
          error
            ? "border-rose-200 bg-rose-50 text-rose-700 hover:bg-rose-100 focus:ring-rose-200"
            : result
              ? "border-emerald-200 bg-emerald-50 text-emerald-700 hover:bg-emerald-100 focus:ring-emerald-200"
              : "border-brand/20 bg-brand/5 text-brand hover:border-brand/35 hover:bg-brand/10 focus:ring-brand/30"
        }`}
      >
        {busy ? (
          <svg aria-hidden="true" viewBox="0 0 24 24" className="h-4 w-4 animate-spin fill-none stroke-current" strokeWidth="2">
            <path strokeLinecap="round" d="M20 12a8 8 0 11-2.34-5.66" />
          </svg>
        ) : (
          <svg aria-hidden="true" viewBox="0 0 24 24" className="h-4 w-4 fill-none stroke-current" strokeWidth="1.8">
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 16V4m0 0L7.5 8.5M12 4l4.5 4.5M5 14v4.5A1.5 1.5 0 006.5 20h11a1.5 1.5 0 001.5-1.5V14" />
          </svg>
        )}
        {label}
      </button>
    </>
  );
}
