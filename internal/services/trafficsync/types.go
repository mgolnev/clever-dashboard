package trafficsync

// StatusReport — состояние автоматической загрузки трафика.
type StatusReport struct {
	Enabled bool           `json:"enabled"`
	Syncing bool           `json:"syncing"`
	Sources []SourceStatus `json:"sources"`
}

// TriggerResult — результат неблокирующего ручного запуска синхронизации.
type TriggerResult struct {
	Started bool `json:"started"`
	Syncing bool `json:"syncing"`
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
