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

// Channels — витрины (канал заказа: приложение/сайт) для фильтра.
func (s *Service) Channels() ([]NamedCount, error) {
	return s.repo.channels()
}

// Payments — способы оплаты для фильтра (по убыванию числа заказов).
func (s *Service) Payments() ([]NamedCount, error) {
	return s.repo.payments()
}

// Deliveries — способы доставки для фильтра (по убыванию числа заказов).
func (s *Service) Deliveries() ([]NamedCount, error) {
	return s.repo.deliveries()
}

// Coupons — промокоды (купоны заказа) для фильтра (по убыванию числа заказов).
func (s *Service) Coupons() ([]NamedCount, error) {
	return s.repo.coupons()
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

// Report считает метрики за период [start,end] и за период сравнения с опциональным
// фильтром по городу и/или области. start/end — даты YYYY-MM-DD; если пустые, берётся
// последняя неделя данных. compareStart/compareEnd задают второй период явно.
func (s *Service) Report(start, end, compareStart, compareEnd string, f Filters) (*Report, error) {
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

	cur, err := s.period(st, en, f)
	if err != nil {
		return nil, err
	}
	prev, err := s.period(prevStart, prevEnd, f)
	if err != nil {
		return nil, err
	}

	return &Report{
		Period:   Range{Start: st.Format(dateLayout), End: en.Format(dateLayout), Days: days},
		Previous: Range{Start: prevStart.Format(dateLayout), End: prevEnd.Format(dateLayout), Days: prevDays},
		Current:  cur,
		Prev:     prev,
	}, nil
}

func (s *Service) period(st, en time.Time, f Filters) (PeriodMetrics, error) {
	startTs := st.Format(dateLayout) + " 00:00:00"
	endTs := en.Format(dateLayout) + " 23:59:59"

	var pm PeriodMetrics
	var err error
	if pm.KPI, err = s.repo.kpi(startTs, endTs, f); err != nil {
		return pm, err
	}
	if pm.Funnel, err = s.repo.funnel(startTs, endTs, f); err != nil {
		return pm, err
	}
	if pm.ByChannel, err = s.repo.groupOrders("channel", startTs, endTs, f, 10); err != nil {
		return pm, err
	}
	if pm.ByPayment, err = s.repo.groupOrders("payment_system", startTs, endTs, f, 10); err != nil {
		return pm, err
	}
	if pm.ByDelivery, err = s.repo.groupOrders("delivery_service", startTs, endTs, f, 10); err != nil {
		return pm, err
	}
	if pm.ByRegion, err = s.repo.groupOrders("region", startTs, endTs, f, 10); err != nil {
		return pm, err
	}
	if pm.TopProducts, err = s.repo.groupProducts("name", startTs, endTs, f, 15); err != nil {
		return pm, err
	}
	if pm.ByCategory, err = s.repo.groupProducts("category", startTs, endTs, f, 12); err != nil {
		return pm, err
	}
	if pm.ByGender, err = s.repo.groupProducts("gender", startTs, endTs, f, 6); err != nil {
		return pm, err
	}
	if pm.ByBrand, err = s.repo.groupProducts("brand", startTs, endTs, f, 8); err != nil {
		return pm, err
	}
	if pm.TopCustomers, err = s.repo.topCustomers(startTs, endTs, f, 20); err != nil {
		return pm, err
	}
	if pm.KPI.Revenue > 0 {
		for i := range pm.TopCustomers {
			pm.TopCustomers[i].RevenueShare = round2(float64(pm.TopCustomers[i].Revenue) / float64(pm.KPI.Revenue) * 100)
		}
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
	if pm.TopCustomers == nil {
		pm.TopCustomers = []CustomerRow{}
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
