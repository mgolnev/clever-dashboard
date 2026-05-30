export interface Range {
  start: string;
  end: string;
  days: number;
}

export interface StageKPI {
  orders: number;
  revenue: number;
  units: number;
  aov: number;
  asp: number;
  upt: number;
}

export interface KPIStages {
  created: StageKPI;
  paid: StageKPI;
  inTransit: StageKPI;
  completed: StageKPI;
  terminal: StageKPI;
  paidTerminal: StageKPI;
}

export interface KPI {
  orders: number;
  netOrders: number;
  revenue: number;
  aov: number;
  asp: number;
  paidOrders: number;
  paidRate: number;
  canceledOrders: number;
  canceledRate: number;
  units: number;
  customers: number;
  completed: number;
  terminal: number;
  inTransit: number;
  g2n: number;
  redemptionRate: number;
  stages: KPIStages;
}

export interface NamedCount {
  name: string;
  orders: number;
  revenue: number;
}

export interface FunnelStage {
  stage: string;
  label: string;
  orders: number;
}

export interface ProductRow {
  name: string;
  units: number;
  orders: number;
  revenue: number;
}

export interface PeriodMetrics {
  kpi: KPI;
  funnel: FunnelStage[];
  byChannel: NamedCount[];
  byPayment: NamedCount[];
  byDelivery: NamedCount[];
  byRegion: NamedCount[];
  topProducts: ProductRow[];
  byCategory: ProductRow[];
  byGender: ProductRow[];
  byBrand: ProductRow[];
}

export interface Report {
  period: Range;
  previous: Range;
  current: PeriodMetrics;
  prev: PeriodMetrics;
}

export interface Bounds {
  min: string;
  max: string;
}

export interface City {
  name: string;
  orders: number;
}

export interface FunnelStep {
  key: string;
  label: string;
  orders: number;
  fromStart: number;
  fromPrev: number;
}

export interface SegmentRow {
  name: string;
  gross: number;
  paid: number;
  paidRate: number;
  completed: number;
  completedRate: number;
  canceled: number;
  cancelRate: number;
  problems: number;
  revenue: number;
}

export interface SegmentGroup {
  by: string;
  label: string;
  rows: SegmentRow[];
}

export interface LabeledCount {
  label: string;
  orders: number;
}

export interface FunnelReport {
  period: Range;
  stages: FunnelStep[];
  gross: number;
  canceled: number;
  returns: number;
  problems: number;
  canceledNoReason: number;
  segments: SegmentGroup[];
  topProblems: LabeledCount[];
  topCancelReasons: LabeledCount[];
}

export interface ImportResult {
  importId: number;
  filename: string;
  rowsTotal: number;
  ordersImported: number;
  itemsImported: number;
  periodStart: string | null;
  periodEnd: string | null;
}
