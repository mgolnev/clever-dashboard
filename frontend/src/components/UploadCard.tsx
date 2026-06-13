import { useRef, useState, useEffect, useCallback } from "react";
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
  const [localFiles, setLocalFiles] = useState<string[]>([]);

  const loadLocalFiles = useCallback(async () => {
    try {
      const files = await api.localFiles();
      setLocalFiles(files || []);
    } catch {
      // Игнорируем ошибки, если бэкенд не поддерживает или папка недоступна
    }
  }, []);

  useEffect(() => {
    loadLocalFiles();
  }, [loadLocalFiles]);

  async function upload(file: File) {
    setBusy(true);
    setError(null);
    setResult(null);
    try {
      const res = await api.importFile(file);
      setResult(res);
      onImported();
      loadLocalFiles(); // Обновим список, если там что-то изменилось
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка загрузки");
    } finally {
      setBusy(false);
      if (inputRef.current) inputRef.current.value = "";
    }
  }

  async function importLocal(filename: string) {
    setBusy(true);
    setError(null);
    setResult(null);
    try {
      const res = await api.importLocalFile(filename);
      setResult(res);
      onImported();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка импорта");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="rounded-xl border border-dashed border-slate-300 bg-white p-4 shadow-sm space-y-3">
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
          className="text-sm text-slate-600 file:mr-3 file:rounded-lg file:border-0 file:bg-brand file:px-4 file:py-2 file:text-sm file:font-medium file:text-white hover:file:bg-indigo-700 disabled:opacity-50"
        />
        {busy && <span className="text-sm text-slate-500">Обработка данных…</span>}
        {result && !busy && (
          <span className="text-sm text-emerald-700">
            Успешно импортировано: {num(result.ordersImported)} заказов, {num(result.itemsImported)} позиций
            {result.periodStart && ` · ${result.periodStart.slice(0, 10)} — ${result.periodEnd?.slice(0, 10)}`}
          </span>
        )}
        {error && <span className="text-sm text-rose-600">{error}</span>}
      </div>

      <p className="text-xs text-slate-400">
        Загрузите выгрузку заказов из Битрикса (XLS/HTML или CSV). Повторная загрузка обновляет данные по номеру заказа.
      </p>

      {localFiles.length > 0 && (
        <div className="pt-2 border-t border-slate-100">
          <p className="text-xs font-semibold text-slate-600 mb-1.5">
            Файлы на сервере в папке данных (доступны для мгновенного импорта без ограничений по размеру):
          </p>
          <div className="flex flex-wrap gap-2">
            {localFiles.map((name) => (
              <button
                key={name}
                type="button"
                disabled={busy}
                onClick={() => importLocal(name)}
                className="inline-flex items-center gap-1 rounded bg-slate-100 px-2 py-1 text-xs text-slate-700 hover:bg-slate-200 transition disabled:opacity-50"
              >
                <span>📄</span> {name}
              </button>
            ))}
          </div>
        </div>
      )}

      <div className="rounded bg-slate-50 p-2 text-[11px] text-slate-500">
        💡 <strong>Совет для тяжелых файлов:</strong> Если файл весит больше 10 МБ, облако Amvera может выдать ошибку <em>Payload too large</em> на уровне Nginx. 
        Чтобы обойти это, просто загрузите файл через панель Amvera во вкладку <strong>«Файлы»</strong> (или по SFTP) напрямую в папку данных приложения. После этого файл появится выше и вы сможете импортировать его за секунду.
      </div>
    </div>
  );
}
