// Package logistics — атомарный сервис аналитики доставки для пилота
// «бесплатная доставка / без порога»: KPI, разрезы, когорты пилот/контроль, недельная динамика.
package logistics

import (
	"fmt"
	"time"
)

const dateLayout = "2006-01-02"

type Service struct {
	repo        *Repository
	pilotCities []string
	pilotStart  string
}

func NewService(repo *Repository, pilotCities []string, pilotStart string) *Service {
	return &Service{
		repo:        repo,
		pilotCities: pilotCities,
		pilotStart:  pilotStart,
	}
}

// resolvePrevRange возвращает диапазон для сравнения. Если compareStart и compareEnd
// заданы — используется он (с нормализацией порядка). Иначе — предыдущий период той
// же длины непосредственно перед текущим (поведение по умолчанию).
func resolvePrevRange(st, en time.Time, days int, compareStart, compareEnd string) (time.Time, time.Time, error) {
	if compareStart == "" || compareEnd == "" {
		prevEnd := st.AddDate(0, 0, -1)
		prevStart := prevEnd.AddDate(0, 0, -(days - 1))
		return prevStart, prevEnd, nil
	}
	ps, err := time.Parse(dateLayout, compareStart)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("неверная дата сравнения: %w", err)
	}
	pe, err := time.Parse(dateLayout, compareEnd)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("неверная дата сравнения: %w", err)
	}
	if ps.After(pe) {
		ps, pe = pe, ps
	}
	return ps, pe, nil
}

// Report считает метрики логистики за период и период сравнения.
func (s *Service) Report(start, end, compareStart, compareEnd string, f Filters, granularity string) (*Report, error) {
	start, end, err := s.resolveRange(start, end)
	if err != nil {
		return nil, err
	}
	st, err := time.Parse(dateLayout, start)
	if err != nil {
		return nil, fmt.Errorf("неверная дата начала: %w", err)
	}
	en, err := time.Parse(dateLayout, end)
	if err != nil {
		return nil, fmt.Errorf("неверная дата конца: %w", err)
	}
	if en.Before(st) {
		st, en = en, st
	}
	days := int(en.Sub(st).Hours()/24) + 1
	prevStart, prevEnd, err := resolvePrevRange(st, en, days, compareStart, compareEnd)
	if err != nil {
		return nil, err
	}
	prevDays := int(prevEnd.Sub(prevStart).Hours()/24) + 1

	cur, err := s.period(st, en, f, parseGranularity(granularity))
	if err != nil {
		return nil, err
	}
	prev, err := s.period(prevStart, prevEnd, f, parseGranularity(granularity))
	if err != nil {
		return nil, err
	}

	return &Report{
		Period:      Range{Start: st.Format(dateLayout), End: en.Format(dateLayout), Days: days},
		Previous:    Range{Start: prevStart.Format(dateLayout), End: prevEnd.Format(dateLayout), Days: prevDays},
		Current:     cur,
		Prev:        prev,
		PilotCities: nonNilStrings(s.pilotCities),
		PilotStart:  s.pilotStart,
	}, nil
}

// groupColumn сопоставляет ключ разреза из API колонке заказов.
func groupColumn(groupBy string) string {
	switch groupBy {
	case "city":
		return "city"
	case "region":
		return "region"
	case "delivery":
		return "delivery_service"
	case "payment":
		return "payment_system"
	case "channel":
		return "channel"
	case "coupon":
		return "coupon"
	default:
		return ""
	}
}

// SeriesBreakdown строит динамику в разрезе измерения groupBy за период
// [start,end] с учётом фильтров. Возвращает топ-значения по числу заказов.
func (s *Service) SeriesBreakdown(start, end string, f Filters, groupBy, granularity string) (*SeriesBreakdown, error) {
	col := groupColumn(groupBy)
	if col == "" {
		return nil, fmt.Errorf("неизвестный разрез: %q", groupBy)
	}
	start, end, err := s.resolveRange(start, end)
	if err != nil {
		return nil, err
	}
	st, err := time.Parse(dateLayout, start)
	if err != nil {
		return nil, fmt.Errorf("неверная дата начала: %w", err)
	}
	en, err := time.Parse(dateLayout, end)
	if err != nil {
		return nil, fmt.Errorf("неверная дата конца: %w", err)
	}
	if en.Before(st) {
		st, en = en, st
	}
	days := int(en.Sub(st).Hours()/24) + 1
	startTs := st.Format(dateLayout) + " 00:00:00"
	endTs := en.Format(dateLayout) + " 23:59:59"

	groups, weeks, err := s.repo.seriesBreakdown(startTs, endTs, f, col, 8, parseGranularity(granularity))
	if err != nil {
		return nil, err
	}
	if groups == nil {
		groups = []SeriesGroup{}
	}
	if weeks == nil {
		weeks = []string{}
	}
	return &SeriesBreakdown{
		Period: Range{Start: st.Format(dateLayout), End: en.Format(dateLayout), Days: days},
		Weeks:  weeks,
		Groups: groups,
	}, nil
}

// nonNilStrings гарантирует JSON [] вместо null для пустого списка городов.
func nonNilStrings(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func (s *Service) period(st, en time.Time, f Filters, g Granularity) (PeriodLogistics, error) {
	startTs := st.Format(dateLayout) + " 00:00:00"
	endTs := en.Format(dateLayout) + " 23:59:59"

	var pm PeriodLogistics
	var err error
	if pm.Summary, err = s.repo.summary(startTs, endTs, f); err != nil {
		return pm, err
	}
	if pm.ByService, err = s.repo.byService(startTs, endTs, f, 12); err != nil {
		return pm, err
	}
	pilotSet := make(map[string]bool, len(s.pilotCities))
	for _, c := range s.pilotCities {
		pilotSet[c] = true
	}
	if pm.ByCity, err = s.repo.byCity(startTs, endTs, f, pilotSet, 25); err != nil {
		return pm, err
	}
	if pm.Series, err = s.repo.series(startTs, endTs, f, g); err != nil {
		return pm, err
	}
	if len(s.pilotCities) > 0 {
		pilot, err := s.repo.cohortSummary(startTs, endTs, f, s.pilotCities, true)
		if err != nil {
			return pm, err
		}
		control, err := s.repo.cohortSummary(startTs, endTs, f, s.pilotCities, false)
		if err != nil {
			return pm, err
		}
		pm.Cohorts = &CohortCompare{Pilot: pilot, Control: control}
	}
	pm.ensureNonNil()
	return pm, nil
}

func (pm *PeriodLogistics) ensureNonNil() {
	if pm.ByService == nil {
		pm.ByService = []ServiceRow{}
	}
	if pm.ByCity == nil {
		pm.ByCity = []CityRow{}
	}
	if pm.Series == nil {
		pm.Series = []WeekPoint{}
	}
}

func (s *Service) resolveRange(start, end string) (string, string, error) {
	if start != "" && end != "" {
		return start, end, nil
	}
	min, max, err := s.repo.dataBounds()
	if err != nil {
		return "", "", err
	}
	if max == "" {
		today := time.Now().Format(dateLayout)
		return today, today, nil
	}
	if end == "" {
		end = max
	}
	if start == "" {
		en, _ := time.Parse(dateLayout, end)
		st := en.AddDate(0, 0, -6)
		start = st.Format(dateLayout)
		if min != "" && start < min {
			start = min
		}
	}
	return start, end, nil
}

func parseGranularity(s string) Granularity {
	switch s {
	case string(GranularityDay):
		return GranularityDay
	case string(GranularityMonth):
		return GranularityMonth
	default:
		return GranularityWeek
	}
}
