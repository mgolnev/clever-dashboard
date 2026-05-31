// Package funnel — атомарный сервис анализа пути заказа: кумулятивная воронка
// «гросс → оплата → сборка → отправка → доставка → выкуп» и её разрезы по
// способу оплаты, доставке, каналу и региону, плюс топ проблем и причин отмены.
package funnel

import (
	"fmt"
	"time"
)

const dateLayout = "2006-01-02"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

// Report строит воронку за период [start,end] с опциональным фильтром по городу
// и/или области. Пустые даты — последняя неделя данных.
func (s *Service) Report(start, end string, f Filters) (*Funnel, error) {
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

	c, err := s.repo.reach(startTs, endTs, f)
	if err != nil {
		return nil, err
	}

	stages := s.buildStages(c)

	segs := make([]SegmentGroup, 0, 4)
	for _, def := range []struct{ by, col, label string }{
		{"payment", "payment_system", "Способ оплаты"},
		{"delivery", "delivery_service", "Служба доставки"},
		{"channel", "channel", "Канал заказа"},
		{"region", "region", "Регион"},
	} {
		rows, err := s.repo.segment(def.col, startTs, endTs, f, 12)
		if err != nil {
			return nil, err
		}
		if rows == nil {
			rows = []SegmentRow{}
		}
		segs = append(segs, SegmentGroup{By: def.by, Label: def.label, Rows: rows})
	}

	topProblems, err := s.repo.topLabeled("problem_desc", startTs, endTs, f, 8)
	if err != nil {
		return nil, err
	}
	topReasons, err := s.repo.topLabeled("cancel_reason", startTs, endTs, f, 8)
	if err != nil {
		return nil, err
	}
	if topProblems == nil {
		topProblems = []LabeledCount{}
	}
	if topReasons == nil {
		topReasons = []LabeledCount{}
	}
	reasonsFilled := 0
	for _, r := range topReasons {
		reasonsFilled += r.Orders
	}

	return &Funnel{
		Period:           Range{Start: st.Format(dateLayout), End: en.Format(dateLayout), Days: days},
		Stages:           stages,
		Gross:            c.orders["created"],
		Canceled:         c.canceled,
		Returns:          c.returns,
		Problems:         c.problems,
		CanceledNoReason: c.canceled - reasonsFilled,
		Segments:         segs,
		TopProblems:      topProblems,
		TopCancelReasons: topReasons,
	}, nil
}

func (s *Service) buildStages(c reachCounts) []Stage {
	stages := make([]Stage, 0, len(stageDefs))
	gross := c.orders["created"]
	prev := 0
	for i, def := range stageDefs {
		v := c.orders[def.key]
		st := Stage{
			Key:     def.key,
			Label:   def.label,
			Orders:  v,
			Revenue: c.revenue[def.key],
			Units:   c.units[def.key],
		}
		if gross > 0 {
			st.FromStart = round1(float64(v) / float64(gross) * 100)
		}
		if i == 0 {
			st.FromPrev = 100
		} else if prev > 0 {
			st.FromPrev = round1(float64(v) / float64(prev) * 100)
		}
		stages = append(stages, st)
		prev = v
	}
	return stages
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
		start = en.AddDate(0, 0, -6).Format(dateLayout)
		if min != "" && start < min {
			start = min
		}
	}
	return start, end, nil
}
