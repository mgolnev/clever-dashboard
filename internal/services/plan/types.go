package plan

// ChannelTargets — план NET по каналам (рубли).
type ChannelTargets struct {
	All  int `json:"all"`
	Site int `json:"site"`
	App  int `json:"app"`
}

// PlanMonth — один месяц годового плана.
type PlanMonth struct {
	Month       int            `json:"month"`
	DaysInMonth int            `json:"daysInMonth"`
	Targets     ChannelTargets `json:"targets"`
	PerDay      ChannelTargets `json:"perDay"`
}

// PlanReport — годовой план продаж NET.
type PlanReport struct {
	Year   int         `json:"year"`
	Months []PlanMonth `json:"months"`
}

// PlanItem — элемент для сохранения плана.
type PlanItem struct {
	Month     int    `json:"month"`
	Channel   string `json:"channel"`
	NetTarget int    `json:"netTarget"`
}

// SaveRequest — тело PUT /api/plan.
type SaveRequest struct {
	Year  int        `json:"year"`
	Items []PlanItem `json:"items"`
}
