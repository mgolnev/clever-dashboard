// Package plan — атомарный сервис плана продаж NET по месяцам и каналам.
package plan

import (
	"fmt"
	"math"
	"time"
)

var validChannels = map[string]bool{"all": true, "site": true, "app": true}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func daysInMonth(year, month int) int {
	return time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()
}

func perDay(target, days int) int {
	if days <= 0 {
		return 0
	}
	return int(math.Round(float64(target) / float64(days)))
}

func (s *Service) buildReport(year int, rows []row) *PlanReport {
	type key struct {
		month   int
		channel string
	}
	vals := make(map[key]int)
	for _, rw := range rows {
		vals[key{rw.Month, rw.Channel}] = rw.NetTarget
	}

	months := make([]PlanMonth, 12)
	for m := 1; m <= 12; m++ {
		dim := daysInMonth(year, m)
		targets := ChannelTargets{
			All:  vals[key{m, "all"}],
			Site: vals[key{m, "site"}],
			App:  vals[key{m, "app"}],
		}
		months[m-1] = PlanMonth{
			Month:       m,
			DaysInMonth: dim,
			Targets:     targets,
			PerDay: ChannelTargets{
				All:  perDay(targets.All, dim),
				Site: perDay(targets.Site, dim),
				App:  perDay(targets.App, dim),
			},
		}
	}
	return &PlanReport{Year: year, Months: months}
}

// Get возвращает план за год (все 12 месяцев, нули если не задано).
func (s *Service) Get(year int) (*PlanReport, error) {
	if year < 2000 || year > 2100 {
		return nil, fmt.Errorf("год должен быть в диапазоне 2000..2100")
	}
	rows, err := s.repo.loadYear(year)
	if err != nil {
		return nil, err
	}
	return s.buildReport(year, rows), nil
}

func validateItem(it PlanItem) error {
	if it.Month < 1 || it.Month > 12 {
		return fmt.Errorf("месяц %d вне диапазона 1..12", it.Month)
	}
	if !validChannels[it.Channel] {
		return fmt.Errorf("неизвестный канал %q", it.Channel)
	}
	if it.NetTarget < 0 {
		return fmt.Errorf("netTarget не может быть отрицательным")
	}
	return nil
}

// Save сохраняет элементы плана и возвращает обновлённый отчёт.
func (s *Service) Save(year int, items []PlanItem) (*PlanReport, error) {
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
