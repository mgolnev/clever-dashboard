// Package metrics — атомарный сервис аналитики: KPI, воронка, срезы и
// сравнение периода с предыдущим периодом той же длины.
package metrics

import (
	"fmt"
	"time"
)

const dateLayout = "2006-01-02"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

// DataBounds — доступный диапазон дат заказов (для инициализации UI).
func (s *Service) DataBounds() (string, string, error) {
	return s.repo.dataBounds()
}

// Cities — города для фильтра (по убыванию числа заказов).
func (s *Service) Cities() ([]NamedCount, error) {
	return s.repo.cities()
}

// Regions — области/регионы для фильтра (по убыванию числа заказов).
func (s *Service) Regions() ([]NamedCount, error) {
	return s.repo.regions()
}

// Report считает метрики за период [start,end] и за предыдущий период такой же
// длины с опциональным фильтром по городу и/или области. start/end — даты
// YYYY-MM-DD; если пустые, берётся последняя неделя данных.
func (s *Service) Report(start, end, city, region string) (*Report, error) {
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

	prevEnd := st.AddDate(0, 0, -1)
	prevStart := prevEnd.AddDate(0, 0, -(days - 1))

	cur, err := s.period(st, en, city, region)
	if err != nil {
		return nil, err
	}
	prev, err := s.period(prevStart, prevEnd, city, region)
	if err != nil {
		return nil, err
	}

	return &Report{
		Period:   Range{Start: st.Format(dateLayout), End: en.Format(dateLayout), Days: days},
		Previous: Range{Start: prevStart.Format(dateLayout), End: prevEnd.Format(dateLayout), Days: days},
		Current:  cur,
		Prev:     prev,
	}, nil
}

func (s *Service) period(st, en time.Time, city, region string) (PeriodMetrics, error) {
	startTs := st.Format(dateLayout) + " 00:00:00"
	endTs := en.Format(dateLayout) + " 23:59:59"

	var pm PeriodMetrics
	var err error
	if pm.KPI, err = s.repo.kpi(startTs, endTs, city, region); err != nil {
		return pm, err
	}
	if pm.Funnel, err = s.repo.funnel(startTs, endTs, city, region); err != nil {
		return pm, err
	}
	if pm.ByChannel, err = s.repo.groupOrders("channel", startTs, endTs, city, region, 10); err != nil {
		return pm, err
	}
	if pm.ByPayment, err = s.repo.groupOrders("payment_system", startTs, endTs, city, region, 10); err != nil {
		return pm, err
	}
	if pm.ByDelivery, err = s.repo.groupOrders("delivery_service", startTs, endTs, city, region, 10); err != nil {
		return pm, err
	}
	if pm.ByRegion, err = s.repo.groupOrders("region", startTs, endTs, city, region, 10); err != nil {
		return pm, err
	}
	if pm.TopProducts, err = s.repo.groupProducts("name", startTs, endTs, city, region, 15); err != nil {
		return pm, err
	}
	if pm.ByCategory, err = s.repo.groupProducts("category", startTs, endTs, city, region, 12); err != nil {
		return pm, err
	}
	if pm.ByGender, err = s.repo.groupProducts("gender", startTs, endTs, city, region, 6); err != nil {
		return pm, err
	}
	if pm.ByBrand, err = s.repo.groupProducts("brand", startTs, endTs, city, region, 8); err != nil {
		return pm, err
	}
	pm.ensureNonNil()
	return pm, nil
}

// ensureNonNil заменяет nil-срезы пустыми, чтобы JSON отдавал [] вместо null
// (фронтенд вызывает .map по этим полям).
func (pm *PeriodMetrics) ensureNonNil() {
	if pm.Funnel == nil {
		pm.Funnel = []FunnelStage{}
	}
	for _, p := range []*[]NamedCount{&pm.ByChannel, &pm.ByPayment, &pm.ByDelivery, &pm.ByRegion} {
		if *p == nil {
			*p = []NamedCount{}
		}
	}
	for _, p := range []*[]ProductRow{&pm.TopProducts, &pm.ByCategory, &pm.ByGender, &pm.ByBrand} {
		if *p == nil {
			*p = []ProductRow{}
		}
	}
}

// resolveRange подставляет дефолт (последние 7 дней данных), если даты не заданы.
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
