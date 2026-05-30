import { useCallback, useEffect, useState } from "react";
import { api } from "./api";
import type { Bounds, City, FunnelReport, Report } from "./types";
import DateRangeBar from "./components/DateRangeBar";
import UploadCard from "./components/UploadCard";
import KpiCards from "./components/KpiCards";
import Funnel from "./components/Funnel";
import BreakdownList from "./components/BreakdownList";
import ProductTable from "./components/ProductTable";
import FunnelTab from "./components/FunnelTab";

type Tab = "overview" | "funnels";

function addDays(date: string, days: number): string {
  const d = new Date(date + "T00:00:00");
  d.setDate(d.getDate() + days);
  return d.toISOString().slice(0, 10);
}

export default function App() {
  const [bounds, setBounds] = useState<Bounds | null>(null);
  const [start, setStart] = useState("");
  const [end, setEnd] = useState("");
  const [report, setReport] = useState<Report | null>(null);
  const [funnel, setFunnel] = useState<FunnelReport | null>(null);
  const [cities, setCities] = useState<City[]>([]);
  const [city, setCity] = useState("");
  const [tab, setTab] = useState<Tab>("overview");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadBounds = useCallback(async () => {
    const b = await api.bounds();
    setBounds(b);
    if (b.max && (!start || !end)) {
      setEnd(b.max);
      setStart(addDays(b.max, -6));
    }
    return b;
  }, [start, end]);

  const loadCities = useCallback(async () => {
    const list = await api.cities().catch(() => [] as City[]);
    setCities(list);
  }, []);

  useEffect(() => {
    loadBounds().catch((e) => setError(e.message));
    loadCities();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!start || !end) return;
    setLoading(true);
    setError(null);
    Promise.all([api.metrics(start, end, city), api.funnel(start, end, city)])
      .then(([m, f]) => {
        setReport(m);
        setFunnel(f);
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [start, end, city]);

  const onChange = (s: string, e: string) => {
    setStart(s);
    setEnd(e);
  };

  const onImported = async () => {
    const b = await loadBounds().catch(() => null);
    if (b?.max) {
      setEnd(b.max);
      setStart(addDays(b.max, -6));
    }
    loadCities();
  };

  const hasData = !!bounds?.max;

  return (
    <div className="mx-auto max-w-7xl px-4 py-6">
      <header className="mb-5">
        <h1 className="text-2xl font-bold text-ink">CLEVER Dashboard</h1>
        <p className="text-sm text-slate-500">
          Недельный обзор интернет-магазина CleverWear.ru · источник: Битрикс
        </p>
      </header>

      <div className="mb-4">
        <UploadCard onImported={onImported} />
      </div>

      {!hasData && !loading && (
        <div className="rounded-xl bg-white p-8 text-center text-slate-500 shadow-sm ring-1 ring-slate-200">
          Данных пока нет. Загрузите выгрузку заказов из Битрикса, чтобы увидеть дашборд.
        </div>
      )}

      {hasData && (
        <>
          <div className="mb-4 flex gap-1 border-b border-slate-200">
            {(
              [
                ["overview", "Обзор"],
                ["funnels", "Воронки"],
              ] as [Tab, string][]
            ).map(([key, label]) => (
              <button
                key={key}
                onClick={() => setTab(key)}
                className={`-mb-px border-b-2 px-4 py-2 text-sm font-medium transition ${
                  tab === key
                    ? "border-brand text-brand"
                    : "border-transparent text-slate-500 hover:text-slate-700"
                }`}
              >
                {label}
              </button>
            ))}
          </div>
          <div className="mb-4">
            <DateRangeBar
              start={start}
              end={end}
              min={bounds?.min}
              max={bounds?.max}
              previous={tab === "overview" ? report?.previous : undefined}
              cities={cities}
              city={city}
              onCityChange={setCity}
              onChange={onChange}
            />
          </div>
        </>
      )}

      {error && (
        <div className="mb-4 rounded-lg bg-rose-50 p-3 text-sm text-rose-700">{error}</div>
      )}

      {loading && <div className="py-4 text-sm text-slate-400">Загрузка метрик…</div>}

      {tab === "funnels" && funnel && <FunnelTab report={funnel} />}

      {tab === "overview" && report && (
        <div className="space-y-4">
          <KpiCards current={report.current.kpi} prev={report.prev.kpi} />

          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <Funnel stages={report.current.funnel} />
            <BreakdownList title="Канал заказа (приложение / сайт)" rows={report.current.byChannel} />
          </div>

          <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
            <BreakdownList title="Способы оплаты" rows={report.current.byPayment} />
            <BreakdownList title="Службы доставки" rows={report.current.byDelivery} />
            <BreakdownList title="Топ регионов" rows={report.current.byRegion} />
          </div>

          <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
            <ProductTable title="По категориям" rows={report.current.byCategory} />
            <ProductTable title="По полу" rows={report.current.byGender} />
            <ProductTable title="По брендам" rows={report.current.byBrand} />
          </div>

          <ProductTable title="Топ товаров по выручке" rows={report.current.topProducts} />
        </div>
      )}
    </div>
  );
}
