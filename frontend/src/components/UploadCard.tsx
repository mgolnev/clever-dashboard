import { useRef, useState } from "react";
import { api } from "../api";
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

  async function upload(file: File) {
    setBusy(true);
    setError(null);
    try {
      const res = await api.importFile(file);
      setResult(res);
      onImported();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка загрузки");
    } finally {
      setBusy(false);
      if (inputRef.current) inputRef.current.value = "";
    }
  }

  return (
    <div className="rounded-xl border border-dashed border-slate-300 bg-white p-4 shadow-sm">
      <div className="flex flex-wrap items-center gap-3">
        <input
          ref={inputRef}
          type="file"
          accept=".xls,.xlsx,.csv,.html,.htm"
          disabled={busy}
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) upload(f);
          }}
          className="text-sm text-slate-600 file:mr-3 file:rounded-lg file:border-0 file:bg-brand file:px-4 file:py-2 file:text-sm file:font-medium file:text-white hover:file:bg-indigo-700"
        />
        {busy && <span className="text-sm text-slate-500">Загрузка и обработка…</span>}
        {result && !busy && (
          <span className="text-sm text-emerald-700">
            Загружено: {num(result.ordersImported)} заказов, {num(result.itemsImported)} позиций
            {result.periodStart && ` · ${result.periodStart.slice(0, 10)} — ${result.periodEnd?.slice(0, 10)}`}
          </span>
        )}
        {error && <span className="text-sm text-rose-600">{error}</span>}
      </div>
      <p className="mt-2 text-xs text-slate-400">
        Загрузите выгрузку заказов из Битрикса (XLS/HTML или CSV). Повторная загрузка обновляет данные по номеру заказа.
      </p>
    </div>
  );
}
