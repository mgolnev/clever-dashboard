// Package traffic — атомарный сервис учёта визитов по каналам с приоритетом
// авто-источников (Метрика/AppMetrica) над ручным вводом.
package traffic

import (
	"fmt"
)

var validChannels = map[string]bool{"site": true, "app": true}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

// better сообщает, приоритетнее ли кандидат a текущего лучшего b: сначала по
// источнику (авто > ручной), при равенстве — по свежести updated_at.
func better(a, b row) bool {
	pa, pb := sourcePriority[a.Source], sourcePriority[b.Source]
	if pa != pb {
		return pa > pb
	}
	return a.UpdatedAt > b.UpdatedAt
}

func (s *Service) buildReport(year int, rows []row) *TrafficReport {
	type key struct {
		month   int
		channel string
	}
	best := make(map[key]row)
	for _, rw := range rows {
		k := key{rw.Month, rw.Channel}
		if cur, ok := best[k]; !ok || better(rw, cur) {
			best[k] = rw
		}
	}

	months := make([]TrafficMonth, 12)
	for m := 1; m <= 12; m++ {
		site := best[key{m, "site"}]
		app := best[key{m, "app"}]
		months[m-1] = TrafficMonth{
			Month:      m,
			Site:       site.Visits,
			App:        app.Visits,
			SiteSource: site.Source,
			AppSource:  app.Source,
		}
	}
	return &TrafficReport{Year: year, Months: months}
}

// Get возвращает трафик за год (все 12 месяцев) с приоритетом авто-источников.
func (s *Service) Get(year int) (*TrafficReport, error) {
	if year < 2000 || year > 2100 {
		return nil, fmt.Errorf("год должен быть в диапазоне 2000..2100")
	}
	rows, err := s.repo.loadYear(year)
	if err != nil {
		return nil, err
	}
	return s.buildReport(year, rows), nil
}

func validateItem(it TrafficItem) error {
	if it.Month < 1 || it.Month > 12 {
		return fmt.Errorf("месяц %d вне диапазона 1..12", it.Month)
	}
	if !validChannels[it.Channel] {
		return fmt.Errorf("неизвестный канал %q", it.Channel)
	}
	if !validSource(it.Source) {
		return fmt.Errorf("неизвестный источник %q", it.Source)
	}
	if it.Visits < 0 {
		return fmt.Errorf("visits не может быть отрицательным")
	}
	return nil
}

// Save сохраняет элементы трафика и возвращает обновлённый отчёт.
// Пустой источник элемента трактуется как ручной ввод.
func (s *Service) Save(year int, items []TrafficItem) (*TrafficReport, error) {
	if year < 2000 || year > 2100 {
		return nil, fmt.Errorf("год должен быть в диапазоне 2000..2100")
	}
	for i := range items {
		if items[i].Source == "" {
			items[i].Source = SourceManual
		}
		if err := validateItem(items[i]); err != nil {
			return nil, err
		}
	}
	if len(items) > 0 {
		if err := s.repo.upsert(year, items); err != nil {
			return nil, err
		}
	}
	return s.Get(year)
}
