package traffic

// Источники данных трафика. Авто-источники имеют приоритет над ручным вводом.
const (
	SourceManual     = "manual"     // ручной ввод в дашборде
	SourceMetrika    = "metrika"    // Яндекс.Метрика — визиты сайта
	SourceAppMetrica = "appmetrica" // AppMetrica — визиты приложения
)

// sourcePriority задаёт приоритет источника при разрешении конфликтов за
// один (год, месяц, канал): чем больше значение, тем выше приоритет.
// Авто-источники перекрывают ручной ввод (fallback).
var sourcePriority = map[string]int{
	SourceManual:     0,
	SourceMetrika:    1,
	SourceAppMetrica: 1,
}

func validSource(s string) bool {
	_, ok := sourcePriority[s]
	return ok
}

// TrafficMonth — визиты за месяц по каналам с указанием победившего источника.
type TrafficMonth struct {
	Month      int    `json:"month"`
	Site       int    `json:"site"`
	App        int    `json:"app"`
	SiteSource string `json:"siteSource,omitempty"`
	AppSource  string `json:"appSource,omitempty"`
}

// TrafficReport — годовой отчёт по трафику.
type TrafficReport struct {
	Year   int            `json:"year"`
	Months []TrafficMonth `json:"months"`
}

// TrafficItem — элемент для сохранения трафика. Пустой Source трактуется как ручной ввод.
type TrafficItem struct {
	Month   int    `json:"month"`
	Channel string `json:"channel"`
	Visits  int    `json:"visits"`
	Source  string `json:"source,omitempty"`
}

// SaveRequest — тело PUT /api/traffic.
type SaveRequest struct {
	Year  int           `json:"year"`
	Items []TrafficItem `json:"items"`
}
