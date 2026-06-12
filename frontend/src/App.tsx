import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "./api";
import type { Bounds, City, FunnelReport, LogisticsReport, Range, Report } from "./types";
import DateRangeBar from "./components/DateRangeBar";
import UploadCard from "./components/UploadCard";
import KpiCards from "./components/KpiCards";
import Funnel from "./components/Funnel";
import CustomerKpiCards from "./components/CustomerKpiCards";
import CustomerTable from "./components/CustomerTable";
import BreakdownList from "./components/BreakdownList";
import ProductTable from "./components/ProductTable";
import FunnelTab from "./components/FunnelTab";
import LogisticsTab from "./components/LogisticsTab";
import DynamicsTab from "./components/DynamicsTab";
import PlanTab from "./components/PlanTab";

type Tab = "overview" | "plan" | "customers" | "funnels" | "logistics" | "dynamics";
type CompareMode = "prev" | "yoy" | "prevMonth" | "custom";

// formatLocalDate форматирует дату в YYYY-MM-DD по локальным компонентам,
// без UTC-сдвига (toISOString в зонах восточнее UTC уводит дату на день назад).
function formatLocalDate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

function addDays(date: string, days: number): string {
  const d = new Date(date + "T00:00:00");
  d.setDate(d.getDate() + days);
  return formatLocalDate(d);
}

function shiftYears(date: string, n: number): string {
  const d = new Date(date + "T00:00:00");
  d.setFullYear(d.getFullYear() + n);
  return formatLocalDate(d);
}

function shiftMonths(date: string, n: number): string {
  const d = new Date(date + "T00:00:00");
  d.setMonth(d.getMonth() + n);
  return formatLocalDate(d);
}

export default function App() {
  const [bounds, setBounds] = useState<Bounds | null>(null);
  const [start, setStart] = useState("");
  const [end, setEnd] = useState("");
  const [compareEnabled, setCompareEnabled] = useState(true);
  const [compareMode, setCompareMode] = useState<CompareMode>("prev");
  const [compareStart, setCompareStart] = useState("");
  const [compareEnd, setCompareEnd] = useState("");
  const [report, setReport] = useState<Report | null>(null);
  const [funnel, setFunnel] = useState<FunnelReport | null>(null);
  const [logistics, setLogistics] = useState<LogisticsReport | null>(null);
  const [cities, setCities] = useState<City[]>([]);
  const [city, setCity] = useState<string[]>([]);
  const [regions, setRegions] = useState<City[]>([]);
  const [region, setRegion] = useState<string[]>([]);
  const [channels, setChannels] = useState<City[]>([]);
  const [channel, setChannel] = useState<string[]>([]);
  const [payments, setPayments] = useState<City[]>([]);
  const [payment, setPayment] = useState<string[]>([]);
  const [deliveries, setDeliveries] = useState<City[]>([]);
  const [delivery, setDelivery] = useState<string[]>([]);
  const [coupons, setCoupons] = useState<City[]>([]);
  const [coupon, setCoupon] = useState<string[]>([]);
  const [tab, setTab] = useState<Tab>("overview");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const effectiveCompare = useMemo(() => {
    if (!compareEnabled) return { cs: "", ce: "" };
    switch (compareMode) {
      case "prev":
        return { cs: "", ce: "" };
      case "yoy":
        return start && end ? { cs: shiftYears(start, -1), ce: shiftYears(end, -1) } : { cs: "", ce: "" };
      case "prevMonth":
        return start && end ? { cs: shiftMonths(start, -1), ce: shiftMonths(end, -1) } : { cs: "", ce: "" };
      case "custom":
        return compareStart && compareEnd ? { cs: compareStart, ce: compareEnd } : { cs: "", ce: "" };
      default:
        return { cs: "", ce: "" };
    }
  }, [compareEnabled, compareMode, start, end, compareStart, compareEnd]);

  const { cs, ce } = effectiveCompare;

  const previousForBar: Range | undefined = useMemo(() => {
    if (!compareEnabled) return undefined;
    if (tab === "overview" || tab === "customers") return report?.previous;
    if (tab === "funnels") return funnel?.previous;
    if (tab === "logistics" || tab === "dynamics") return logistics?.previous;
    return undefined;
  }, [compareEnabled, tab, report?.previous, funnel?.previous, logistics?.previous]);

  const loadBounds = useCallback(async () => {
    const b = await api.bounds();
    setBounds(b);
    if (b.max && (!start || !end)) {
      setEnd(b.max);
      setStart(addDays(b.max, -6));
    }
    return b;
  }, [start, end]);

  const loadGeo = useCallback(async () => {
    const [cs, rs, ch, pm, dl, cp] = await Promise.all([
      api.cities().catch(() => [] as City[]),
      api.regions().catch(() => [] as City[]),
      api.channels().catch(() => [] as City[]),
      api.payments().catch(() => [] as City[]),
      api.deliveries().catch(() => [] as City[]),
      api.coupons().catch(() => [] as City[]),
    ]);
    setCities(cs);
    setRegions(rs);
    setChannels(ch);
    setPayments(pm);
    setDeliveries(dl);
    setCoupons(cp);
  }, []);

  useEffect(() => {
    loadBounds().catch((e) => setError(e.message));
    loadGeo();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const cityKey = city.join(",");
  const regionKey = region.join(",");
  const channelKey = channel.join(",");
  const paymentKey = payment.join(",");
  const deliveryKey = delivery.join(",");
  const couponKey = coupon.join(",");

  useEffect(() => {
    if (!start || !end) return;
    setLoading(true);
    setError(null);
    const f = { city, region, channel, payment, delivery, coupon };
    Promise.all([
      api.metrics(start, end, f, cs, ce),
      api.funnel(start, end, f, cs, ce),
      api.logistics(start, end, f, undefined, cs, ce),
    ])
      .then(([m, fn, l]) => {
        setReport(m);
        setFunnel(fn);
        setLogistics(l);
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [start, end, cs, ce, cityKey, regionKey, channelKey, paymentKey, deliveryKey, couponKey]);

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
    loadGeo();
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

      <div className="mb-4 flex gap-1 border-b border-slate-200">
        {(
          [
            ["overview", "Обзор"],
            ["plan", "Цель"],
            ["customers", "Клиенты"],
            ["funnels", "Воронки"],
            ["logistics", "Логистика"],
            ["dynamics", "Динамика"],
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

      {tab === "plan" && <PlanTab />}

      {!hasData && !loading && tab !== "plan" && (
        <div className="rounded-xl bg-white p-8 text-center text-slate-500 shadow-sm ring-1 ring-slate-200">
          Данных пока нет. Загрузите выгрузку заказов из Битрикса, чтобы увидеть дашборд.
        </div>
      )}

      {hasData && tab !== "plan" && (
        <>
          <div className="mb-4">
            <DateRangeBar
              start={start}
              end={end}
              min={bounds?.min}
              max={bounds?.max}
              previous={previousForBar}
              compareEnabled={compareEnabled}
              onToggleCompare={setCompareEnabled}
              compareMode={compareMode}
              onCompareModeChange={setCompareMode}
              compareStart={compareStart}
              compareEnd={compareEnd}
              onCompareRangeChange={(s, e) => {
                setCompareStart(s);
                setCompareEnd(e);
              }}
              cities={cities}
              city={city}
              onCityChange={setCity}
              regions={regions}
              region={region}
              onRegionChange={setRegion}
              channels={channels}
              channel={channel}
              onChannelChange={setChannel}
              payments={payments}
              payment={payment}
              onPaymentChange={setPayment}
              deliveries={deliveries}
              delivery={delivery}
              onDeliveryChange={setDelivery}
              coupons={coupons}
              coupon={coupon}
              onCouponChange={setCoupon}
              onChange={onChange}
            />
          </div>
        </>
      )}

      {error && (
        <div className="mb-4 rounded-lg bg-rose-50 p-3 text-sm text-rose-700">{error}</div>
      )}

      {loading && tab !== "plan" && (
        <div className="py-4 text-sm text-slate-400">Загрузка метрик…</div>
      )}

      {tab !== "plan" && tab === "funnels" && funnel && (
        <FunnelTab report={funnel} showCompare={compareEnabled} />
      )}

      {tab !== "plan" && tab === "logistics" && !loading && !logistics && (
        <div className="rounded-xl bg-white p-6 text-center text-sm text-slate-500 shadow-sm ring-1 ring-slate-200">
          Не удалось загрузить метрики логистики. Проверьте, что backend перезапущен и отвечает на{" "}
          <code className="text-xs">/api/logistics</code>.
        </div>
      )}

      {tab !== "plan" && tab === "logistics" && logistics && (
        <LogisticsTab report={logistics} showCompare={compareEnabled} />
      )}

      {tab !== "plan" && tab === "dynamics" && !loading && !logistics && (
        <div className="rounded-xl bg-white p-6 text-center text-sm text-slate-500 shadow-sm ring-1 ring-slate-200">
          Не удалось загрузить данные динамики. Проверьте, что backend перезапущен и отвечает на{" "}
          <code className="text-xs">/api/logistics</code>.
        </div>
      )}

      {tab !== "plan" && tab === "dynamics" && logistics && (
        <DynamicsTab
          report={logistics}
          start={start}
          end={end}
          filters={{ city, region, channel, payment, delivery, coupon }}
          compareStart={cs}
          compareEnd={ce}
          showCompare={compareEnabled}
        />
      )}

      {tab !== "plan" && tab === "customers" && report && (
        <div className="space-y-4">
          <CustomerKpiCards
            current={report.current.kpi}
            prev={report.prev.kpi}
            showCompare={compareEnabled}
          />
          <CustomerTable rows={report.current.topCustomers} totalRevenue={report.current.kpi.revenue} />
        </div>
      )}

      {tab !== "plan" && tab === "overview" && report && (
        <div className="space-y-4">
          <KpiCards current={report.current.kpi} prev={report.prev.kpi} showCompare={compareEnabled} />

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
