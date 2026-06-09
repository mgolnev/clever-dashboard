package logistics

// Granularity — шаг группировки временного ряда (день / неделя / месяц).
type Granularity string

const (
	GranularityDay   Granularity = "day"
	GranularityWeek  Granularity = "week"
	GranularityMonth Granularity = "month"
)

// Range — границы периода (даты YYYY-MM-DD).
type Range struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Days  int    `json:"days"`
}

// Summary — агрегаты по доставке за период (все гросс-заказы периода).
type Summary struct {
	Orders           int     `json:"orders"`
	Revenue          int     `json:"revenue"`
	PaidOrders       int     `json:"paidOrders"`
	PaidRate         float64 `json:"paidRate"`
	DeliveryTotal    int     `json:"deliveryTotal"`
	AvgDelivery      int     `json:"avgDelivery"`
	FreeOrders       int     `json:"freeOrders"`
	FreeDeliveryRate float64 `json:"freeDeliveryRate"`
}

// ServiceRow — разрез по службе доставки.
type ServiceRow struct {
	Name             string  `json:"name"`
	Orders           int     `json:"orders"`
	Share            float64 `json:"share"`
	PaidOrders       int     `json:"paidOrders"`
	PaidRate         float64 `json:"paidRate"`
	Revenue          int     `json:"revenue"`
	DeliveryTotal    int     `json:"deliveryTotal"`
	AvgDelivery      int     `json:"avgDelivery"`
	FreeOrders       int     `json:"freeOrders"`
	FreeDeliveryRate float64 `json:"freeDeliveryRate"`
}

// CityRow — разрез по городу.
type CityRow struct {
	Name             string  `json:"name"`
	IsPilot          bool    `json:"isPilot"`
	Orders           int     `json:"orders"`
	Share            float64 `json:"share"`
	PaidOrders       int     `json:"paidOrders"`
	PaidRate         float64 `json:"paidRate"`
	Revenue          int     `json:"revenue"`
	DeliveryTotal    int     `json:"deliveryTotal"`
	AvgDelivery      int     `json:"avgDelivery"`
	FreeOrders       int     `json:"freeOrders"`
	FreeDeliveryRate float64 `json:"freeDeliveryRate"`
}

// CohortCompare — пилотные города vs остальные (в рамках фильтра области).
type CohortCompare struct {
	Pilot   Summary `json:"pilot"`
	Control Summary `json:"control"`
}

// WeekPoint — недельная динамика по всем основным метрикам.
type WeekPoint struct {
	Week             string  `json:"week"`
	Orders           int     `json:"orders"`
	NetOrders        int     `json:"netOrders"`
	PaidOrders       int     `json:"paidOrders"`
	Revenue          int     `json:"revenue"`
	Units            int     `json:"units"`
	AOV              int     `json:"aov"`
	ASP              int     `json:"asp"`
	UPT              float64 `json:"upt"`
	PaidRate         float64 `json:"paidRate"`
	AvgDelivery      int     `json:"avgDelivery"`
	FreeDeliveryRate float64 `json:"freeDeliveryRate"`
	DeliveryTotal    int     `json:"deliveryTotal"`
}

// SeriesGroup — недельная динамика одного значения разреза (город/служба/...).
// Points выровнены по общему списку недель SeriesBreakdown.Weeks (пропуски — нули).
type SeriesGroup struct {
	Name   string      `json:"name"`
	Points []WeekPoint `json:"points"`
}

// SeriesBreakdown — недельная динамика в разрезе измерения (для вкладки «Динамика»).
type SeriesBreakdown struct {
	Period Range         `json:"period"`
	Weeks  []string      `json:"weeks"`
	Groups []SeriesGroup `json:"groups"`
}

// PeriodLogistics — метрики за один период.
type PeriodLogistics struct {
	Summary   Summary        `json:"summary"`
	ByService []ServiceRow   `json:"byService"`
	ByCity    []CityRow      `json:"byCity"`
	Cohorts   *CohortCompare `json:"cohorts,omitempty"`
	Series    []WeekPoint    `json:"series"`
}

// Report — текущий и предыдущий период + настройки пилота.
type Report struct {
	Period      Range           `json:"period"`
	Previous    Range           `json:"previous"`
	Current     PeriodLogistics `json:"current"`
	Prev        PeriodLogistics `json:"prev"`
	PilotCities []string        `json:"pilotCities"`
	PilotStart  string          `json:"pilotStart,omitempty"`
}
