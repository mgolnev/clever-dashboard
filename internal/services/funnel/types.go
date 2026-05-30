package funnel

// Range — границы периода (YYYY-MM-DD).
type Range struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Days  int    `json:"days"`
}

// Stage — стадия кумулятивной воронки (заказ «дошёл хотя бы до»).
type Stage struct {
	Key       string  `json:"key"`
	Label     string  `json:"label"`
	Orders    int     `json:"orders"`
	FromStart float64 `json:"fromStart"` // % от гросс
	FromPrev  float64 `json:"fromPrev"`  // % шага от предыдущей стадии
}

// SegmentRow — воронка в разрезе одного значения справочника.
type SegmentRow struct {
	Name          string  `json:"name"`
	Gross         int     `json:"gross"`
	Paid          int     `json:"paid"`
	PaidRate      float64 `json:"paidRate"`      // paid / gross
	Completed     int     `json:"completed"`
	CompletedRate float64 `json:"completedRate"` // completed / gross
	Canceled      int     `json:"canceled"`
	CancelRate    float64 `json:"cancelRate"`    // canceled / gross
	Problems      int     `json:"problems"`
	Revenue       int     `json:"revenue"`
}

// SegmentGroup — набор строк воронки по одному разрезу (оплата/доставка/канал...).
type SegmentGroup struct {
	By    string       `json:"by"`
	Label string       `json:"label"`
	Rows  []SegmentRow `json:"rows"`
}

// LabeledCount — строка для топа проблем/причин отмены.
type LabeledCount struct {
	Label  string `json:"label"`
	Orders int    `json:"orders"`
}

// Funnel — полный отчёт воронки за период.
type Funnel struct {
	Period           Range          `json:"period"`
	Stages           []Stage        `json:"stages"`
	Gross            int            `json:"gross"`
	Canceled         int            `json:"canceled"`
	Returns          int            `json:"returns"`
	Problems         int            `json:"problems"`
	CanceledNoReason int            `json:"canceledNoReason"`
	Segments         []SegmentGroup `json:"segments"`
	TopProblems      []LabeledCount `json:"topProblems"`
	TopCancelReasons []LabeledCount `json:"topCancelReasons"`
}

// стадии воронки и их подписи.
var stageDefs = []struct{ key, label, predicate string }{
	{"created", "Создан (гросс)", ""},
	{"paid", "Оплачен", "paid"},
	{"processing", "В сборке/обработке", "processing"},
	{"shipped", "Отправлен", "shipped"},
	{"delivered", "Доставлен в ПВЗ", "delivered"},
	{"completed", "Выполнен (выкуп)", "completed"},
}
