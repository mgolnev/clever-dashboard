package goal

// Bounds — доступный диапазон дат заказов.
type Bounds struct {
	Min string `json:"min"`
	Max string `json:"max"`
}

// ChannelTargets — месячная цель NET по каналам.
type ChannelTargets struct {
	All  int `json:"all"`
	Site int `json:"site"`
	App  int `json:"app"`
}

type PlanMonth struct {
	Month       int            `json:"month"`
	DaysInMonth int            `json:"daysInMonth"`
	Targets     ChannelTargets `json:"targets"`
	PerDay      ChannelTargets `json:"perDay"`
}

type PlanReport struct {
	Year   int         `json:"year"`
	Months []PlanMonth `json:"months"`
}

// ChannelSummary содержит только показатели, необходимые вкладке «Цель».
type ChannelSummary struct {
	Channel    string  `json:"channel"`
	Revenue    int     `json:"revenue"`
	NetRevenue int     `json:"netRevenue"`
	AOV        int     `json:"aov"`
	Sessions   int     `json:"sessions"`
	NetCR      float64 `json:"netCr"`
}

type PeriodSummary struct {
	Start    string           `json:"start"`
	End      string           `json:"end"`
	Channels []ChannelSummary `json:"channels"`
}

// SourceDefinition описывает настроенный источник без зависимости от trafficsync.
type SourceDefinition struct {
	Source     string
	Channel    string
	Configured bool
}

type SourceStatus struct {
	Source       string `json:"source"`
	Channel      string `json:"channel"`
	Configured   bool   `json:"configured"`
	Status       string `json:"status"`
	DateFrom     string `json:"dateFrom,omitempty"`
	DateTo       string `json:"dateTo,omitempty"`
	RowsImported int    `json:"rowsImported"`
	Error        string `json:"error,omitempty"`
	StartedAt    string `json:"startedAt,omitempty"`
	FinishedAt   string `json:"finishedAt,omitempty"`
	LastDataDay  string `json:"lastDataDay,omitempty"`
}

type AnalyticsStatus struct {
	Enabled bool           `json:"enabled"`
	Sources []SourceStatus `json:"sources"`
}

// Report — компактная read-модель вкладки «Цель».
type Report struct {
	Year            int             `json:"year"`
	Month           int             `json:"month"`
	Plan            PlanReport      `json:"plan"`
	Bounds          Bounds          `json:"bounds"`
	Current         PeriodSummary   `json:"current"`
	History         *PeriodSummary  `json:"history"`
	AnalyticsStatus AnalyticsStatus `json:"analyticsStatus"`
}
