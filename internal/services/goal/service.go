// Package goal формирует компактную read-модель вкладки «Цель».
package goal

import (
	"fmt"
	"math"
	"time"
)

type Options struct {
	AnalyticsEnabled bool
	Sources          []SourceDefinition
}

type Service struct {
	repo    *Repository
	options Options
}

func NewService(repo *Repository, options Options) *Service {
	return &Service{repo: repo, options: options}
}

func (s *Service) Report(year, month int) (*Report, error) {
	if year < 2000 || year > 2100 {
		return nil, fmt.Errorf("год должен быть в диапазоне 2000..2100")
	}
	if month < 1 || month > 12 {
		return nil, fmt.Errorf("месяц должен быть в диапазоне 1..12")
	}

	bounds, err := s.repo.bounds()
	if err != nil {
		return nil, err
	}
	planRows, err := s.repo.plan(year)
	if err != nil {
		return nil, err
	}
	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)
	current, err := s.period(monthStart, monthEnd)
	if err != nil {
		return nil, err
	}

	var history *PeriodSummary
	if historyStart, historyEnd, ok := historyRange(monthStart, bounds); ok {
		value, periodErr := s.period(historyStart, historyEnd)
		if periodErr != nil {
			return nil, periodErr
		}
		history = &value
	}
	status, err := s.analyticsStatus()
	if err != nil {
		return nil, err
	}

	return &Report{
		Year:            year,
		Month:           month,
		Plan:            buildPlan(year, planRows),
		Bounds:          bounds,
		Current:         current,
		History:         history,
		AnalyticsStatus: status,
	}, nil
}

func (s *Service) period(start, end time.Time) (PeriodSummary, error) {
	startValue, endValue := start.Format("2006-01-02"), end.Format("2006-01-02")
	orders, err := s.repo.orders(startValue, endValue)
	if err != nil {
		return PeriodSummary{}, err
	}
	sessions, err := s.repo.sessions(startValue, endValue)
	if err != nil {
		return PeriodSummary{}, err
	}
	allConversionOrders := orders.Channels["site"].NetOrders + orders.Channels["app"].NetOrders
	allSessions := sessions["site"] + sessions["app"]
	return PeriodSummary{
		Start: startValue,
		End:   endValue,
		Channels: []ChannelSummary{
			makeChannel("all", orders.All, allConversionOrders, allSessions),
			makeChannel("site", orders.Channels["site"], orders.Channels["site"].NetOrders, sessions["site"]),
			makeChannel("app", orders.Channels["app"], orders.Channels["app"].NetOrders, sessions["app"]),
		},
	}, nil
}

func makeChannel(channel string, orders orderAggregate, conversionOrders, sessions int) ChannelSummary {
	summary := ChannelSummary{
		Channel:    channel,
		Revenue:    orders.Revenue,
		NetRevenue: orders.RedeemedGrossRevenue - orders.RefundAmount,
		Sessions:   sessions,
	}
	if orders.NetOrders > 0 {
		summary.AOV = orders.Revenue / orders.NetOrders
	}
	if sessions > 0 {
		summary.NetCR = round2(float64(conversionOrders) / float64(sessions) * 100)
	}
	return summary
}

func historyRange(monthStart time.Time, bounds Bounds) (time.Time, time.Time, bool) {
	if bounds.Max == "" {
		return time.Time{}, time.Time{}, false
	}
	maxDate, err := time.Parse("2006-01-02", bounds.Max)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	end := monthStart.AddDate(0, 0, -1)
	if maxDate.Before(end) {
		end = maxDate
	}
	if bounds.Min != "" {
		minDate, parseErr := time.Parse("2006-01-02", bounds.Min)
		if parseErr != nil || end.Before(minDate) {
			return time.Time{}, time.Time{}, false
		}
		start := end.AddDate(0, 0, -29)
		if start.Before(minDate) {
			start = minDate
		}
		return start, end, true
	}
	return end.AddDate(0, 0, -29), end, true
}

func buildPlan(year int, rows []planRow) PlanReport {
	type key struct {
		month   int
		channel string
	}
	values := make(map[key]int, len(rows))
	for _, row := range rows {
		values[key{month: row.Month, channel: row.Channel}] = row.NetTarget
	}
	months := make([]PlanMonth, 12)
	for month := 1; month <= 12; month++ {
		days := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()
		targets := ChannelTargets{
			All:  values[key{month: month, channel: "all"}],
			Site: values[key{month: month, channel: "site"}],
			App:  values[key{month: month, channel: "app"}],
		}
		months[month-1] = PlanMonth{
			Month:       month,
			DaysInMonth: days,
			Targets:     targets,
			PerDay: ChannelTargets{
				All:  perDay(targets.All, days),
				Site: perDay(targets.Site, days),
				App:  perDay(targets.App, days),
			},
		}
	}
	return PlanReport{Year: year, Months: months}
}

func perDay(target, days int) int {
	if days <= 0 {
		return 0
	}
	return int(math.Round(float64(target) / float64(days)))
}

func (s *Service) analyticsStatus() (AnalyticsStatus, error) {
	runs, err := s.repo.latestRuns()
	if err != nil {
		return AnalyticsStatus{}, err
	}
	days, err := s.repo.latestDataDays()
	if err != nil {
		return AnalyticsStatus{}, err
	}
	status := AnalyticsStatus{Enabled: s.options.AnalyticsEnabled, Sources: make([]SourceStatus, 0, len(s.options.Sources))}
	for _, source := range s.options.Sources {
		item := SourceStatus{
			Source: source.Source, Channel: source.Channel, Configured: source.Configured,
			Status: "never", LastDataDay: days[source.Source],
		}
		if run, ok := runs[source.Source]; ok {
			item.Status = run.Status
			item.DateFrom = run.DateFrom
			item.DateTo = run.DateTo
			item.RowsImported = run.RowsImported
			item.Error = run.Error
			item.StartedAt = run.StartedAt
			item.FinishedAt = run.FinishedAt
		}
		status.Sources = append(status.Sources, item)
	}
	return status, nil
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
