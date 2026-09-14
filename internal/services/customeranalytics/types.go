package customeranalytics

// Filters — сквозные фильтры заказов. Значения каждого поля передаются через
// запятую и объединяются по ИЛИ; разные поля объединяются по И.
type Filters struct {
	City     string
	Region   string
	Channel  string
	Payment  string
	Delivery string
	Coupon   string
}

type Range struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Granularity string

const (
	GranularityMonth   Granularity = "month"
	GranularityQuarter Granularity = "quarter"
)

type RetentionCell struct {
	Offset    int     `json:"offset"`
	Month     string  `json:"month"`
	Customers int     `json:"customers"`
	Rate      float64 `json:"rate"`
	Available bool    `json:"available"`
	Complete  bool    `json:"complete"`
}

type Cohort struct {
	Month     string          `json:"month"`
	Customers int             `json:"customers"`
	Cells     []RetentionCell `json:"cells"`
}

// RetentionPoint — взвешенный retention по всем когортам, которые успели
// прожить соответствующее число полных календарных месяцев.
type RetentionPoint struct {
	Offset            int     `json:"offset"`
	Customers         int     `json:"customers"`
	EligibleCustomers int     `json:"eligibleCustomers"`
	Rate              float64 `json:"rate"`
}

type Summary struct {
	Customers       int              `json:"customers"`
	CohortCount     int              `json:"cohortCount"`
	RepeatCustomers int              `json:"repeatCustomers"`
	RepeatRate      float64          `json:"repeatRate"`
	Retention       []RetentionPoint `json:"retention"`
}

// CompositionPeriod раскладывает уникальных клиентов периода на три
// взаимоисключающие группы относительно предыдущего календарного периода.
type CompositionPeriod struct {
	Period       string  `json:"period"`
	Customers    int     `json:"customers"`
	New          int     `json:"new"`
	Active       int     `json:"active"`
	Returned     int     `json:"returned"`
	NewRate      float64 `json:"newRate"`
	ActiveRate   float64 `json:"activeRate"`
	ReturnedRate float64 `json:"returnedRate"`
	Complete     bool    `json:"complete"`
}

type View struct {
	Mode        string              `json:"mode"`
	Summary     Summary             `json:"summary"`
	Cohorts     []Cohort            `json:"cohorts"`
	Composition []CompositionPeriod `json:"composition"`
}

type Report struct {
	Period      Range       `json:"period"`
	Granularity Granularity `json:"granularity"`
	Gross       View        `json:"gross"`
	Paid        View        `json:"paid"`
}
