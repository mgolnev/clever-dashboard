package traffic

// TrafficMonth — визиты за месяц по каналам (ручной ввод).
type TrafficMonth struct {
	Month int `json:"month"`
	Site  int `json:"site"`
	App   int `json:"app"`
}

// TrafficReport — годовой отчёт по трафику.
type TrafficReport struct {
	Year   int            `json:"year"`
	Months []TrafficMonth `json:"months"`
}

// TrafficItem — элемент для сохранения трафика.
type TrafficItem struct {
	Month   int    `json:"month"`
	Channel string `json:"channel"`
	Visits  int    `json:"visits"`
}

// SaveRequest — тело PUT /api/traffic.
type SaveRequest struct {
	Year  int           `json:"year"`
	Items []TrafficItem `json:"items"`
}
