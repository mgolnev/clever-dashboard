// Package acquisition — атомарный сервис верхней воронки: трафик и конверсия
// в заказ по каналам. Он читает общую БД, не вызывая traffic или metrics.
package acquisition

import (
	"fmt"
	"time"
)

const dateLayout = "2006-01-02"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Report(start, end, compareStart, compareEnd string) (*Report, error) {
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
	if st.After(en) {
		st, en = en, st
	}
	days := int(en.Sub(st).Hours()/24) + 1
	ps, pe, err := resolvePrevRange(st, en, days, compareStart, compareEnd)
	if err != nil {
		return nil, err
	}
	cur, err := s.period(st, en)
	if err != nil {
		return nil, err
	}
	prev, err := s.period(ps, pe)
	if err != nil {
		return nil, err
	}
	return &Report{
		Period:   Range{Start: st.Format(dateLayout), End: en.Format(dateLayout), Days: days},
		Previous: Range{Start: ps.Format(dateLayout), End: pe.Format(dateLayout), Days: int(pe.Sub(ps).Hours()/24) + 1},
		Current:  cur, Prev: prev,
	}, nil
}

func (s *Service) period(st, en time.Time) (PeriodData, error) {
	start, end := st.Format(dateLayout), en.Format(dateLayout)
	totals, err := s.repo.trafficTotals(start, end)
	if err != nil {
		return PeriodData{}, err
	}
	if err := s.repo.orderTotals(start, end, totals); err != nil {
		return PeriodData{}, err
	}
	if err := s.repo.ecommerceTotals(start, end, totals); err != nil {
		return PeriodData{}, err
	}
	days, err := s.repo.dailyTraffic(start, end)
	if err != nil {
		return PeriodData{}, err
	}
	if err := s.repo.dailyOrders(start, end, days); err != nil {
		return PeriodData{}, err
	}

	channels := make([]ChannelMetrics, 0, 3)
	all := channelTotals{}
	for _, key := range []string{"site", "app"} {
		t := totals[key]
		all.Sessions += t.Sessions
		all.Users += t.Users
		all.Orders += t.Orders
		all.PaidOrders += t.PaidOrders
		all.NetOrders += t.NetOrders
		all.Sampled = all.Sampled || t.Sampled
		all.ProductViewUsers += t.ProductViewUsers
		all.AddToCartUsers += t.AddToCartUsers
		all.TrackedPurchaseUsers += t.TrackedPurchaseUsers
		all.EcommerceRows += t.EcommerceRows
		label := "Сайт"
		if key == "app" {
			label = "Приложение"
		}
		channels = append(channels, makeChannel(key, label, t))
	}
	channels = append([]ChannelMetrics{makeChannel("all", "Итого", all)}, channels...)

	daily := make([]DailyPoint, 0, int(en.Sub(st).Hours()/24)+1)
	for day := st; !day.After(en); day = day.AddDate(0, 0, 1) {
		key := day.Format(dateLayout)
		daily = append(daily, DailyPoint{
			Day:            key,
			SiteSessions:   days[key]["site"].Sessions,
			AppSessions:    days[key]["app"].Sessions,
			SiteUsers:      days[key]["site"].Users,
			AppUsers:       days[key]["app"].Users,
			SiteOrders:     days[key]["site"].Orders,
			AppOrders:      days[key]["app"].Orders,
			SitePaidOrders: days[key]["site"].PaidOrders,
			AppPaidOrders:  days[key]["app"].PaidOrders,
		})
	}
	return PeriodData{Channels: channels, Daily: daily, HasTraffic: all.Sessions > 0, Sampled: all.Sampled}, nil
}

func makeChannel(channel, label string, t channelTotals) ChannelMetrics {
	m := ChannelMetrics{
		Channel: channel, Label: label, Sessions: t.Sessions, Users: t.Users, Orders: t.Orders,
		PaidOrders: t.PaidOrders, NetOrders: t.NetOrders,
		EcommerceAvailable:   t.EcommerceRows > 0,
		TrackedPurchaseUsers: t.TrackedPurchaseUsers,
	}
	if t.Sessions > 0 {
		m.OrderCR = round2(float64(t.Orders) / float64(t.Sessions) * 100)
		m.PaidCR = round2(float64(t.PaidOrders) / float64(t.Sessions) * 100)
		m.NetCR = round2(float64(t.NetOrders) / float64(t.Sessions) * 100)
	}
	m.EcommerceFunnel = makeEcommerceFunnel(channel, t)
	return m
}

func makeEcommerceFunnel(channel string, t channelTotals) []EcommerceStage {
	stages := []EcommerceStage{
		{Key: "product_view", Label: "Просмотрели товар", Count: t.ProductViewUsers, Unit: "пользователи", FromPrevious: 100},
		{Key: "add_to_cart", Label: "Добавили в корзину", Count: t.AddToCartUsers, Unit: "пользователи", FromPrevious: conversion(t.AddToCartUsers, t.ProductViewUsers)},
	}
	previous := t.AddToCartUsers
	if channel == "app" && t.BeginCheckoutUsers > 0 {
		stages = append(stages, EcommerceStage{
			Key: "begin_checkout", Label: "Начали оформление", Count: t.BeginCheckoutUsers,
			Unit: "пользователи", FromPrevious: conversion(t.BeginCheckoutUsers, previous),
		})
		previous = t.BeginCheckoutUsers
	}
	createdBase := 100.0
	stages = append(stages, EcommerceStage{
		Key: "created", Label: "Созданные заказы", Count: t.Orders, Unit: "заказы",
		FromPrevious: conversion(t.Orders, previous), FromCreated: &createdBase,
	})
	paidFromCreated := conversion(t.PaidOrders, t.Orders)
	stages = append(stages, EcommerceStage{
		Key: "paid", Label: "Оплачено", Count: t.PaidOrders, Unit: "заказы",
		FromPrevious: paidFromCreated, FromCreated: &paidFromCreated,
	})
	return stages
}

func conversion(value, base int) float64 {
	if base <= 0 {
		return 0
	}
	return round2(float64(value) / float64(base) * 100)
}

func resolvePrevRange(st, en time.Time, days int, compareStart, compareEnd string) (time.Time, time.Time, error) {
	if compareStart == "" || compareEnd == "" {
		pe := st.AddDate(0, 0, -1)
		return pe.AddDate(0, 0, -(days - 1)), pe, nil
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

func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }
