// Package traffic — атомарный сервис учёта визитов по каналам (ручной ввод в v1).
package traffic

import (
	"fmt"
)

var validChannels = map[string]bool{"site": true, "app": true}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) buildReport(year int, rows []row) *TrafficReport {
	type key struct {
		month   int
		channel string
	}
	vals := make(map[key]int)
	for _, rw := range rows {
		vals[key{rw.Month, rw.Channel}] = rw.Visits
	}

	months := make([]TrafficMonth, 12)
	for m := 1; m <= 12; m++ {
		months[m-1] = TrafficMonth{
			Month: m,
			Site:  vals[key{m, "site"}],
			App:   vals[key{m, "app"}],
		}
	}
	return &TrafficReport{Year: year, Months: months}
}

// Get возвращает трафик за год (source=manual, все 12 месяцев).
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
	if it.Visits < 0 {
		return fmt.Errorf("visits не может быть отрицательным")
	}
	return nil
}

// Save сохраняет элементы трафика и возвращает обновлённый отчёт.
func (s *Service) Save(year int, items []TrafficItem) (*TrafficReport, error) {
	if year < 2000 || year > 2100 {
		return nil, fmt.Errorf("год должен быть в диапазоне 2000..2100")
	}
	for _, it := range items {
		if err := validateItem(it); err != nil {
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
