package metrics

// Range — границы периода (даты в формате YYYY-MM-DD).
type Range struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Days  int    `json:"days"`
}

// KPI — ключевые показатели за период.
type KPI struct {
	Orders         int     `json:"orders"`
	NetOrders      int     `json:"netOrders"`
	Revenue        int     `json:"revenue"`
	AOV            int     `json:"aov"`
	ASP            int     `json:"asp"`
	PaidOrders     int     `json:"paidOrders"`
	PaidRate       float64 `json:"paidRate"`
	CanceledOrders int     `json:"canceledOrders"`
	CanceledRate   float64 `json:"canceledRate"`
	Units          int     `json:"units"`
	Customers      int     `json:"customers"`
	Completed      int     `json:"completed"`      // выкуплено (status_stage=completed)
	Terminal       int     `json:"terminal"`       // заказы в конечном статусе
	InTransit      int     `json:"inTransit"`      // заказы «в пути» (не дошли до выкупа/отмены)
	G2N            float64 `json:"g2n"`            // выкуплено / оформлено (гросс), %
	RedemptionRate float64 `json:"redemptionRate"` // выкуплено / заказы в конечном статусе, %
}

// NamedCount — срез по справочнику (канал/оплата/доставка/регион).
type NamedCount struct {
	Name    string `json:"name"`
	Orders  int    `json:"orders"`
	Revenue int    `json:"revenue"`
}

// FunnelStage — стадия воронки статусов.
type FunnelStage struct {
	Stage  string `json:"stage"`
	Label  string `json:"label"`
	Orders int    `json:"orders"`
}

// ProductRow — строка товарной аналитики.
type ProductRow struct {
	Name    string `json:"name"`
	Units   int    `json:"units"`
	Orders  int    `json:"orders"`
	Revenue int    `json:"revenue"`
}

// PeriodMetrics — полный набор метрик за один период.
type PeriodMetrics struct {
	KPI         KPI           `json:"kpi"`
	Funnel      []FunnelStage `json:"funnel"`
	ByChannel   []NamedCount  `json:"byChannel"`
	ByPayment   []NamedCount  `json:"byPayment"`
	ByDelivery  []NamedCount  `json:"byDelivery"`
	ByRegion    []NamedCount  `json:"byRegion"`
	TopProducts []ProductRow  `json:"topProducts"`
	ByCategory  []ProductRow  `json:"byCategory"`
	ByGender    []ProductRow  `json:"byGender"`
	ByBrand     []ProductRow  `json:"byBrand"`
}

// Report — отчёт с текущим и предыдущим периодом для сравнения.
type Report struct {
	Period   Range         `json:"period"`
	Previous Range         `json:"previous"`
	Current  PeriodMetrics `json:"current"`
	Prev     PeriodMetrics `json:"prev"`
}

var stageLabels = map[string]string{
	"new":        "Новый",
	"processing": "В обработке",
	"shipped":    "Отправлен",
	"in_pvz":     "В ПВЗ",
	"completed":  "Выполнен",
	"closed":     "Закрыт",
	"returned":   "Возврат",
	"canceled":   "Отменён",
	"other":      "Прочее",
}

// funnelOrder задаёт порядок стадий в воронке.
var funnelOrder = []string{"new", "processing", "shipped", "in_pvz", "completed", "closed", "returned", "canceled"}
